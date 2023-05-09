package conversion

import (
	"fmt"
	"io"
	"net/http"

	"github.com/zeroisme/pizza-crd/pkg/webhook"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

func Serve(w http.ResponseWriter, req *http.Request) {
	var body []byte
	if req.Body != nil {
		if data, err := io.ReadAll(req.Body); err == nil {
			body = data
		}
	}

	contentType := req.Header.Get("Content-Type")
	serializer := webhook.GetInputSerializer(contentType)

	if serializer == nil {
		msg := fmt.Sprintf("invalid Content-Type header `%s`", contentType)
		klog.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	klog.V(2).Infof("handling request: %v", body)
	obj, gvk, err := serializer.Decode(body, nil, nil)

	if err != nil {
		msg := fmt.Sprintf("failed to deserialize body (%v) with error %v", string(body), err)
		klog.Error(err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	var responseObj runtime.Object
	switch *gvk {
	case apiextensionsv1beta1.SchemeGroupVersion.WithKind("ConversionReview"):
		convertReview, ok := obj.(*apiextensionsv1beta1.ConversionReview)
		if !ok {
			msg := fmt.Sprintf("Expected a v1beta1.ConversionReview but got: %T", obj)
			klog.Errorf(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		convertReview.Response = doConvertionV1beta1(convertReview)

		klog.V(2).Info(fmt.Sprintf("sending response: %v", convertReview.Response))
		// reset request
		convertReview.Request = &apiextensionsv1beta1.ConversionRequest{}
		responseObj = convertReview
	case apiextensionsv1.SchemeGroupVersion.WithKind("ConversionReview"):
		convertReview, ok := obj.(*apiextensionsv1.ConversionReview)
		if !ok {
			msg := fmt.Sprintf("Expected v1.ConversionReview but got: %T", obj)
			klog.Errorf(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		convertReview.Response = doConvertionV1(convertReview)
		klog.V(2).Info(fmt.Sprintf("sending response: %v", convertReview.Response))

		// reset the request
		convertReview.Request = &apiextensionsv1.ConversionRequest{}
		responseObj = convertReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		klog.Error(err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	webhook.SendResponse(w, req, responseObj)
}

func doConvertionV1beta1(review *apiextensionsv1beta1.ConversionReview) *apiextensionsv1beta1.ConversionResponse {
	resp := &apiextensionsv1beta1.ConversionResponse{
		Result: metav1.Status{
			Status: metav1.StatusSuccess,
		},
		UID: review.Request.UID,
	}

	converted, err := convert2Desired(review.Request.Objects, review.Request.DesiredAPIVersion)
	if err != nil {
		resp.Result = metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		}
		return resp
	}
	resp.ConvertedObjects = converted
	return resp
}

func doConvertionV1(review *apiextensionsv1.ConversionReview) *apiextensionsv1.ConversionResponse {
	resp := &apiextensionsv1.ConversionResponse{
		Result: metav1.Status{
			Status: metav1.StatusSuccess,
		},
		UID: review.Request.UID,
	}

	converted, err := convert2Desired(review.Request.Objects, review.Request.DesiredAPIVersion)
	if err != nil {
		resp.Result = metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		}
		return resp
	}
	resp.ConvertedObjects = converted
	return resp
}

func convert2Desired(rawObjects []runtime.RawExtension, desiredAPIVersion string) ([]runtime.RawExtension, error) {
	var objs []runtime.Object
	var err error
	for _, in := range rawObjects {
		if in.Object == nil {
			in.Object, _, err = webhook.Codecs.UniversalDeserializer().Decode(in.Raw, nil, nil)
		}
		if err != nil {
			return nil, err
		}
		obj, err := convert(in.Object, desiredAPIVersion)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}

	var results []runtime.RawExtension

	for _, obj := range objs {
		results = append(results, runtime.RawExtension{Object: obj})
	}

	return results, nil
}
