package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"

	"github.com/spf13/pflag"
	"github.com/zeroisme/pizza-crd/pkg/generated/clientset/versioned"
	"github.com/zeroisme/pizza-crd/pkg/webhook/admission"
	"github.com/zeroisme/pizza-crd/pkg/webhook/conversion"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/cli/globalflag"

	restaurantinformers "github.com/zeroisme/pizza-crd/pkg/generated/informers/externalversions"
)

func NewDefaultOptions() *Options {
	o := &Options{
		*options.NewSecureServingOptions(),
	}
	o.SecureServing.ServerCert.PairName = "pizza-crd-webhook"
	return o
}

type Options struct {
	SecureServing options.SecureServingOptions
}

type Config struct {
	SecureServing *server.SecureServingInfo
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.SecureServing.AddFlags(fs)
}

func (o *Options) Config() (*Config, error) {
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("0.0.0.0", nil, nil); err != nil {
		return nil, err
	}

	c := &Config{}

	if err := o.SecureServing.ApplyTo(&c.SecureServing); err != nil {
		return nil, err
	}

	return c, nil
}

func main() {
	opt := NewDefaultOptions()
	fs := pflag.NewFlagSet("pizza-crd-webhook", pflag.ExitOnError)
	globalflag.AddGlobalFlags(fs, "pizza-crd-webhook")
	opt.AddFlags(fs)
	if err := fs.Parse(os.Args); err != nil {
		panic(err)
	}

	cfg, err := opt.Config()
	if err != nil {
		panic(err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		kubeconfig := filepath.Join(home, ".kube", "config")
		if envvar := os.Getenv("KUBECONFIG"); len(envvar) > 0 {
			kubeconfig = envvar
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	}
	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	stopCh := server.SetupSignalHandler()

	// register handlers

	restaurantInformers := restaurantinformers.NewSharedInformerFactory(clientset, time.Second*30)
	mux := http.NewServeMux()
	mux.Handle("/convert/v1beta1/pizza", http.HandlerFunc(conversion.Serve))
	mux.Handle("/admit/v1beta1/pizza", http.HandlerFunc(admission.ServePizzaAdmit))
	mux.Handle("/validate/v1beta1/pizza", http.HandlerFunc(admission.ServePizzaValidation(restaurantInformers)))
	restaurantInformers.Start(stopCh)

	// run server
	if stoppedCh, listenerStoppedCh, err := cfg.SecureServing.Serve(handlers.LoggingHandler(os.Stdout, mux), time.Second*30, stopCh); err != nil {
		panic(err)
	} else {
		<-stoppedCh
		<-listenerStoppedCh
	}
}
