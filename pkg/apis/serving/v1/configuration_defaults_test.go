/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/apis"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"
	"knative.dev/serving/pkg/apis/config"
	"knative.dev/serving/pkg/apis/serving"
	cconfig "knative.dev/serving/pkg/reconciler/configuration/config"
)

const someTimeoutSeconds = 400

func TestConfigurationDefaulting(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		in   *Configuration
		want *Configuration
	}{{
		name: "empty",
		in:   &Configuration{},
		want: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						TimeoutSeconds:       ptr.Int64(config.DefaultRevisionTimeoutSeconds),
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
	}, {
		name: "run latest, not create",
		in: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: "busybox",
							}},
						},
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:           config.DefaultUserContainerName,
								Image:          "busybox",
								Resources:      defaultResources,
								ReadinessProbe: defaultProbe,
							}},
						},
						TimeoutSeconds:       ptr.Int64(config.DefaultRevisionTimeoutSeconds),
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
	}, {
		name: "run latest, create",
		in: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: "busybox",
							}},
						},
					},
				},
			},
		},
		ctx: apis.WithinCreate(context.Background()),
		want: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							EnableServiceLinks: ptr.Bool(false),
							Containers: []corev1.Container{{
								Name:           config.DefaultUserContainerName,
								Image:          "busybox",
								Resources:      defaultResources,
								ReadinessProbe: defaultProbe,
							}},
						},
						TimeoutSeconds:       ptr.Int64(config.DefaultRevisionTimeoutSeconds),
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
	}, {
		name: "run latest with some default overrides",
		in: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							EnableServiceLinks: ptr.Bool(true),
							Containers: []corev1.Container{{
								Image: "busybox",
							}},
						},
						TimeoutSeconds:       ptr.Int64(60),
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							EnableServiceLinks: ptr.Bool(true),
							Containers: []corev1.Container{{
								Name:           config.DefaultUserContainerName,
								Image:          "busybox",
								Resources:      defaultResources,
								ReadinessProbe: defaultProbe,
							}},
						},
						TimeoutSeconds:       ptr.Int64(60),
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
	}, {
		name: "run latest with identical timeout defaults",
		in: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							EnableServiceLinks: ptr.Bool(true),
							Containers: []corev1.Container{{
								Image: "busybox",
							}},
						},
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						PodSpec: corev1.PodSpec{
							EnableServiceLinks: ptr.Bool(true),
							Containers: []corev1.Container{{
								Name:           config.DefaultUserContainerName,
								Image:          "busybox",
								Resources:      defaultResources,
								ReadinessProbe: defaultProbe,
							}},
						},
						TimeoutSeconds:       ptr.Int64(someTimeoutSeconds),
						ContainerConcurrency: ptr.Int64(config.DefaultContainerConcurrency),
					},
				},
			},
		},
		ctx: defaultConfigurationContextWithStore(logtesting.TestLogger(t), corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: config.FeaturesConfigName}},
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: config.DefaultsConfigName,
				},
				Data: map[string]string{
					"revision-timeout-seconds":                strconv.Itoa(someTimeoutSeconds),
					"revision-response-start-timeout-seconds": strconv.Itoa(someTimeoutSeconds),
					"revision-idle-timeout-seconds":           strconv.Itoa(someTimeoutSeconds),
				},
			})(context.Background()),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.in
			ctx := context.Background()
			if test.ctx != nil {
				ctx = test.ctx
			}
			got.SetDefaults(ctx)
			if !cmp.Equal(got, test.want, ignoreUnexportedResources) {
				t.Errorf("SetDefaults (-want, +got) = %v",
					cmp.Diff(test.want, got, ignoreUnexportedResources))
			}
		})
	}
}

func TestBYORevisionName(t *testing.T) {
	new := &Configuration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "thing",
			Annotations: map[string]string{"annotated": "yes"},
		},
		Spec: ConfigurationSpec{
			Template: RevisionTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "thing-2022",
				},
				Spec: RevisionSpec{
					PodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "busybox",
						}},
					},
				},
			},
		},
	}

	old := new.DeepCopy()
	old.ObjectMeta.Annotations = map[string]string{}

	want := new.DeepCopy()

	ctx := apis.WithinUpdate(context.Background(), old)
	new.SetDefaults(ctx)

	if diff := cmp.Diff(want, new); diff != "" {
		t.Errorf("SetDefaults (-want, +got) = %v", diff)
	}

	new.SetDefaults(context.Background())
	if cmp.Equal(want, new, ignoreUnexportedResources) {
		t.Errorf("Expected diff, got none! object: %v", new)
	}
}

func TestConfigurationUserInfo(t *testing.T) {
	const (
		u1 = "oveja@knative.dev"
		u2 = "cabra@knative.dev"
		u3 = "vaca@knative.dev"
	)
	withUserAnns := func(u1, u2 string, s *Configuration) *Configuration {
		a := s.GetAnnotations()
		if a == nil {
			a = map[string]string{}
			s.SetAnnotations(a)
		}
		a[serving.CreatorAnnotation] = u1
		a[serving.UpdaterAnnotation] = u2
		return s
	}
	tests := []struct {
		name     string
		user     string
		this     *Configuration
		prev     *Configuration
		wantAnns map[string]string
	}{{
		name: "create-new",
		user: u1,
		this: &Configuration{},
		prev: nil,
		wantAnns: map[string]string{
			serving.CreatorAnnotation: u1,
			serving.UpdaterAnnotation: u1,
		},
	}, {
		// Old objects don't have the annotation, and unless there's a change in
		// data they won't get it.
		name:     "update-no-diff-old-object",
		user:     u1,
		this:     &Configuration{},
		prev:     &Configuration{},
		wantAnns: map[string]string{},
	}, {
		name: "update-no-diff-new-object",
		user: u2,
		this: withUserAnns(u1, u1, &Configuration{}),
		prev: withUserAnns(u1, u1, &Configuration{}),
		wantAnns: map[string]string{
			serving.CreatorAnnotation: u1,
			serving.UpdaterAnnotation: u1,
		},
	}, {
		name: "update-diff-old-object",
		user: u2,
		this: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						ContainerConcurrency: ptr.Int64(1),
					},
				},
			},
		},
		prev: &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						ContainerConcurrency: ptr.Int64(2),
					},
				},
			},
		},
		wantAnns: map[string]string{
			serving.UpdaterAnnotation: u2,
		},
	}, {
		name: "update-diff-new-object",
		user: u3,
		this: withUserAnns(u1, u2, &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						ContainerConcurrency: ptr.Int64(1),
					},
				},
			},
		}),
		prev: withUserAnns(u1, u2, &Configuration{
			Spec: ConfigurationSpec{
				Template: RevisionTemplateSpec{
					Spec: RevisionSpec{
						ContainerConcurrency: ptr.Int64(2),
					},
				},
			},
		}),
		wantAnns: map[string]string{
			serving.CreatorAnnotation: u1,
			serving.UpdaterAnnotation: u3,
		},
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := apis.WithUserInfo(context.Background(), &authv1.UserInfo{
				Username: test.user,
			})
			if test.prev != nil {
				ctx = apis.WithinUpdate(ctx, test.prev)
				test.prev.SetDefaults(ctx)
			}
			test.this.SetDefaults(ctx)
			if got, want := test.this.GetAnnotations(), test.wantAnns; !cmp.Equal(got, want) {
				t.Errorf("Annotations = %v, want: %v, diff (-got, +want): %s", got, want, cmp.Diff(got, want))
			}
		})
	}
}

func defaultConfigurationContextWithStore(logger *zap.SugaredLogger, cms ...corev1.ConfigMap) func(ctx context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		s := cconfig.NewStore(logger)
		for _, cm := range cms {
			s.OnConfigChanged(&cm)
		}
		return s.ToContext(ctx)
	}
}
