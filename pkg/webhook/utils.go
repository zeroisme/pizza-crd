package webhook

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/munnerz/goautoneg"
	"github.com/zeroisme/pizza-crd/pkg/apis/restaurant/install"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

type mediaType struct {
	Type, SubType string
}

var (
	Scheme      = runtime.NewScheme()
	Codecs      = serializer.NewCodecFactory(Scheme)
	serializers = map[mediaType]runtime.Serializer{
		{"application", "json"}: json.NewSerializer(json.DefaultMetaFactory, Scheme, Scheme, false),
		{"application", "yaml"}: json.NewYAMLSerializer(json.DefaultMetaFactory, Scheme, Scheme),
	}
)

func init() {
	utilruntime.Must(apiextensionsv1.AddToScheme(Scheme))
	utilruntime.Must(apiextensionsv1beta1.AddToScheme(Scheme))

	utilruntime.Must(admissionv1beta1.AddToScheme(Scheme))
	utilruntime.Must(admissionv1.AddToScheme(Scheme))

	install.Install(Scheme)
}

func GetInputSerializer(contentType string) runtime.Serializer {
	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	return serializers[mediaType{parts[0], parts[1]}]
}

func GetOutputSerializer(accept string) runtime.Serializer {
	if len(accept) == 0 {
		return serializers[mediaType{"application", "json"}]
	}

	clauses := goautoneg.ParseAccept(accept)
	for _, clause := range clauses {
		for k, v := range serializers {
			switch {
			case clause.Type == k.Type && clause.SubType == k.SubType,
				clause.Type == k.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				return v
			}
		}
	}
	return nil
}

func SendResponse(w http.ResponseWriter, r *http.Request, obj runtime.Object) {
	var err error
	accept := r.Header.Get("Accept")
	outSerializer := GetOutputSerializer(accept)
	if outSerializer == nil {
		msg := fmt.Sprintf("invalid Accept header `%s`", accept)
		klog.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err = outSerializer.Encode(obj, w)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
