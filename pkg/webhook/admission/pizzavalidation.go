package admission

import (
	"fmt"
	"io"
	"net/http"

	"github.com/zeroisme/pizza-crd/pkg/apis/restaurant/v1alpha1"
	"github.com/zeroisme/pizza-crd/pkg/apis/restaurant/v1beta1"
	restaurantinformers "github.com/zeroisme/pizza-crd/pkg/generated/informers/externalversions"
	restaurantv1alpha1 "github.com/zeroisme/pizza-crd/pkg/generated/listers/restaurant/v1alpha1"
	"github.com/zeroisme/pizza-crd/pkg/webhook"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

func ServePizzaValidation(informers restaurantinformers.SharedInformerFactory) func(http.ResponseWriter, *http.Request) {
	toppingInformer := informers.Restaurant().V1alpha1().Toppings().Informer()
	toppingLister := informers.Restaurant().V1alpha1().Toppings().Lister()

	return func(w http.ResponseWriter, req *http.Request) {
		if !toppingInformer.HasSynced() {
			http.Error(w, "topping informer not synced yet", http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}

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
			review.Response = doValidateV1(review, toppingLister)
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
			review.Response = doValidateV1beta1(review, toppingLister)
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
}

func doValidateV1(review *admissionv1.AdmissionReview, toppingLister restaurantv1alpha1.ToppingLister) *admissionv1.AdmissionResponse {
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
	err = validatePizza(review.Request.Object.Object, toppingLister)
	if err != nil {
		response.Result = &metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		}
		return response
	}
	response.Allowed = true
	response.Result = &metav1.Status{
		Message: "pizza is valid",
		Status:  metav1.StatusSuccess,
	}

	return response
}

func doValidateV1beta1(review *admissionv1beta1.AdmissionReview, toppingLister restaurantv1alpha1.ToppingLister) *admissionv1beta1.AdmissionResponse {
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
	err = validatePizza(review.Request.Object.Object, toppingLister)
	if err != nil {
		response.Result = &metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		}
		return response
	}
	response.Allowed = true
	response.Result = &metav1.Status{
		Message: "pizza is valid",
		Status:  metav1.StatusSuccess,
	}

	return response
}

func validatePizza(pizzaObj runtime.Object, toppingLister restaurantv1alpha1.ToppingLister) error {
	switch pizza := pizzaObj.(type) {
	case *v1alpha1.Pizza:
		for _, topping := range pizza.Spec.Toppings {
			if _, err := toppingLister.Get(topping); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to lookup topping %q: %v", topping, err)
			} else if errors.IsNotFound(err) {
				return fmt.Errorf("topping %q not found", topping)
			}
		}
		return nil
	case *v1beta1.Pizza:
		for _, topping := range pizza.Spec.Toppings {
			if _, err := toppingLister.Get(topping.Name); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to lookup topping %q: %v", topping, err)
			} else if errors.IsNotFound(err) {
				return fmt.Errorf("topping %q not found", topping.Name)
			}
		}
		return nil
	default:
		return fmt.Errorf("unexpected pizza type: %T", pizzaObj)
	}
}
