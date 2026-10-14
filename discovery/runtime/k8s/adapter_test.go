// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestPodToWorkloadLocators(t *testing.T) {
	pod := &corev1.Pod{
		Name: "agent", Namespace: "team-a", UID: "uid-1",
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Ports: []corev1.ContainerPort{{ContainerPort: 9999}}}},
		},
		Status: corev1.PodStatus{PodIP: "10.244.0.9"},
	}
	services := []*corev1.Service{{Name: "svc-a", Namespace: "team-a"}}

	workload := (&adapter{}).podToWorkload(pod, services)

	assert.Equal(t, []*runtimev1.WorkloadLocator{
		{Protocol: "tcp", Url: "tcp://10-244-0-9.team-a.pod:9999"},
		{Protocol: "tcp", Url: "tcp://svc-a.team-a.svc:9999"},
	}, workload.GetLocators())
}
