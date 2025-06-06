package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	netbirdiov1 "github.com/netbirdio/kubernetes-operator/api/v1"
)

var _ = Describe("NBSetupKey Webhook", func() {
	var (
		obj          *netbirdiov1.NBSetupKey
		validator    NBSetupKeyCustomValidator
		resourceName = "test"
		secret       *corev1.Secret
	)

	BeforeEach(func() {
		obj = &netbirdiov1.NBSetupKey{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		}
		validator = NBSetupKeyCustomValidator{
			client: k8sClient,
		}
	})

	Context("When creating or updating NBSetupKey under Validating Webhook", func() {
		When("secretKeyRef is empty", func() {
			It("Should fail", func() {
				obj.Spec = netbirdiov1.NBSetupKeySpec{}
				warnings, err := validator.ValidateCreate(context.Background(), obj)
				Expect(err).To(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		When("secret doesn't exist", func() {
			It("Should fail", func() {
				obj.Spec = netbirdiov1.NBSetupKeySpec{
					SecretKeyRef: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resourceName,
						},
						Key: "setupkey",
					},
				}
				warnings, err := validator.ValidateCreate(context.Background(), obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
			})
		})

		Context("secret exists", Ordered, func() {
			createSecret := func(secretkey, setupkey string) {
				resource := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      resourceName,
					},
					Data: map[string][]byte{
						secretkey: []byte(setupkey),
					},
				}

				secret = &corev1.Secret{}
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: resourceName}, secret)
				if err == nil {
					Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			BeforeEach(func() {
				obj.Spec = netbirdiov1.NBSetupKeySpec{
					SecretKeyRef: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resourceName,
						},
						Key: "setupkey",
					},
				}
			})

			When("secret key doesn't exist", func() {
				It("Should fail", func() {
					createSecret("wrongkey", "EEEEEEEE-EEEE-EEEE-EEEE-EEEEEEEEEEEE")
					warnings, err := validator.ValidateCreate(context.Background(), obj)
					Expect(err).NotTo(HaveOccurred())
					Expect(warnings).NotTo(BeEmpty())
				})
			})
			When("setup key is invalid", func() {
				It("Should fail", func() {
					createSecret("setupkey", "EEEEEEEE")
					warnings, err := validator.ValidateCreate(context.Background(), obj)
					Expect(err).NotTo(HaveOccurred())
					Expect(warnings).NotTo(BeEmpty())
				})
			})
			When("setup key is valid", func() {
				It("Should allow creation", func() {
					createSecret("setupkey", "EEEEEEEE-EEEE-EEEE-EEEE-EEEEEEEEEEEE")
					warnings, err := validator.ValidateCreate(context.Background(), obj)
					Expect(err).NotTo(HaveOccurred())
					Expect(warnings).To(BeEmpty())
				})
			})
		})

		Context("Delete", func() {
			When("No pods exist with annotation", func() {
				BeforeEach(func() {
					pod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "notannotated",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "test"},
							},
						},
					}
					Expect(k8sClient.Create(ctx, pod)).To(Succeed())
				})

				AfterEach(func() {
					pod := &corev1.Pod{}
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: "notannotated"}, pod)
					if !errors.IsNotFound(err) {
						Expect(err).NotTo(HaveOccurred())
						if len(pod.Finalizers) > 0 {
							pod.Finalizers = nil
							Expect(k8sClient.Update(ctx, pod)).To(Succeed())
						}
						err = k8sClient.Delete(ctx, pod)
						if !errors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					}
				})

				It("should allow delete", func() {
					obj = &netbirdiov1.NBSetupKey{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
					}
					Expect(validator.ValidateDelete(ctx, obj)).Error().NotTo(HaveOccurred())
				})
			})
			When("Pods exist with annotation", func() {
				BeforeEach(func() {
					pod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "annotated",
							Annotations: map[string]string{
								setupKeyAnnotation: "test",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "test"},
							},
						},
					}
					Expect(k8sClient.Create(ctx, pod)).To(Succeed())
				})

				AfterEach(func() {
					pod := &corev1.Pod{}
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: "annotated"}, pod)
					if !errors.IsNotFound(err) {
						Expect(err).NotTo(HaveOccurred())
						if len(pod.Finalizers) > 0 {
							pod.Finalizers = nil
							Expect(k8sClient.Update(ctx, pod)).To(Succeed())
						}
						err = k8sClient.Delete(ctx, pod)
						if !errors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					}
				})

				It("should deny delete", func() {
					obj = &netbirdiov1.NBSetupKey{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
					}
					Expect(validator.ValidateDelete(ctx, obj)).Error().To(HaveOccurred())
				})
			})
		})
	})

})
