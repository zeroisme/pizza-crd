package admission

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/appscode/jsonpatch"
	"github.com/zeroisme/pizza-crd/pkg/apis/restaurant/v1alpha1"
	"github.com/zeroisme/pizza-crd/pkg/apis/restaurant/v1beta1"
	"github.com/zeroisme/pizza-crd/pkg/webhook"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

func ServePizzaAdmit(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to read body: %v", err).Error(), http.StatusBadRequest)
	}

	// decode as admission review
	klog.V(2).Infof("handling request: %v", body)
	obj, gvk, err := webhook.Codecs.UniversalDeserializer().Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("failed to deserialize body (%v) with error %v", string(body), err)
		klog.Error(err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object

	switch *gvk {
	case admissionv1.SchemeGroupVersion.WithKind("AdmissionReview"):
		review, ok := obj.(*admissionv1.AdmissionReview)
		if !ok {
			msg := fmt.Sprintf("Expected a v1beta1.AdmissionReview but got: %T", obj)
			klog.Errorf(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		if review.Request == nil {
			msg := "unexpected nil request"
			klog.Errorf(msg)
			http.Error(w, msg, http.StatusBadRequest)
		}
		review.Response = doAdmitV1(review)
		review.Request = &admissionv1.AdmissionRequest{}
		responseObj = review
	case admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview"):
		review, ok := obj.(*admissionv1beta1.AdmissionReview)
		if !ok {
			msg := fmt.Sprintf("Expected a v1beta1.AdmissionReview but got: %T", obj)
			klog.Errorf(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		if review.Request == nil {
			msg := "unexpected nil request"
			klog.Errorf(msg)
			http.Error(w, msg, http.StatusBadRequest)
		}
		review.Response = doAdmitV1beta1(review)
		review.Request = &admissionv1beta1.AdmissionRequest{}
		responseObj = review
	default:
		msg := fmt.Sprintf("unexpected GroupVersionKind: %v", gvk)
		klog.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	webhook.SendResponse(w, req, responseObj)
}

func doAdmitV1(review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	response := &admissionv1.AdmissionResponse{
		UID: review.Request.UID,
	}
	var err error
	if review.Request.Object.Object == nil {
		review.Request.Object.Object, _, err = webhook.Codecs.UniversalDeserializer().Decode(review.Request.Object.Raw, nil, nil)
		if err != nil {
			response.Result = &metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			}
			return response
		}
	}
	orig := review.Request.Object.Raw
	patch, err := patchPizza(orig, review.Request.Object.Object, review.Request.Namespace, review.Request.Name, review.GroupVersionKind())
	if err != nil {
		response.Result = &metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		}
		return response
	}
	response.Patch = patch
	typ := admissionv1.PatchTypeJSONPatch
	response.PatchType = &typ
	response.Allowed = true
	return response
}

func doAdmitV1beta1(review *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	response := &admissionv1beta1.AdmissionResponse{
		UID: review.Request.UID,
	}
	var err error
	if review.Request.Object.Object == nil {
		review.Request.Object.Object, _, err = webhook.Codecs.UniversalDeserializer().Decode(review.Request.Object.Raw, nil, nil)
		if err != nil {
			response.Result = &metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			}
			return response
		}
	}
	orig := review.Request.Object.Raw
	patch, err := patchPizza(orig, review.Request.Object.Object, review.Request.Namespace, review.Request.Name, review.GroupVersionKind())
	if err != nil {
		response.Result = &metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		}
		return response
	}
	response.Patch = patch
	typ := admissionv1beta1.PatchTypeJSONPatch
	response.PatchType = &typ
	response.Allowed = true
	return response
}

func patchPizza(orig []byte, pizza runtime.Object, namespace string, name string, gvk schema.GroupVersionKind) ([]byte, error) {
	bs, err := defaultingPizza(pizza)
	if err != nil {
		return nil, err
	}
	klog.V(2).Infof("Defaulting %s/%s in version %s", namespace, name, gvk)
	ops, err := jsonpatch.CreatePatch(orig, bs)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ops)
}

func defaultingPizza(pizza runtime.Object) ([]byte, error) {
	var bs []byte
	var err error
	switch p := pizza.(type) {
	case *v1alpha1.Pizza:
		// default toppings
		if len(p.Spec.Toppings) == 0 {
			p.Spec.Toppings = []string{"tomato", "mozzarella", "salami"}
		}
		bs, err = json.Marshal(p)
		if err != nil {
			return nil, err
		}
		return bs, nil
	case *v1beta1.Pizza:
		if len(p.Spec.Toppings) == 0 {
			p.Spec.Toppings = []v1beta1.PizzaTopping{
				{Name: "tomato", Quantity: 1},
				{Name: "mozzarella", Quantity: 1},
				{Name: "salami", Quantity: 1},
			}
		}
		bs, err = json.Marshal(p)
		if err != nil {
			return nil, err
		}
		return bs, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", pizza)
	}
}
