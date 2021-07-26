package model

import (
	"testing"

	"github.com/evergreen-ci/evergreen"
	"github.com/evergreen-ci/evergreen/model/pod"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPICreatePod(t *testing.T) {
	t.Run("ToService", func(t *testing.T) {
		apiPod := APICreatePod{
			Name:   utility.ToStringPtr("id"),
			Memory: utility.ToIntPtr(128),
			CPU:    utility.ToIntPtr(128),
			Image:  utility.ToStringPtr("image"),
			EnvVars: []*APIPodEnvVar{
				{
					Name:   utility.ToStringPtr("name"),
					Value:  utility.ToStringPtr("value"),
					Secret: utility.ToBoolPtr(false),
				},
				{
					Name:   utility.ToStringPtr("name1"),
					Value:  utility.ToStringPtr("value1"),
					Secret: utility.ToBoolPtr(false),
				},
				{
					Name:   utility.ToStringPtr("secret_name"),
					Value:  utility.ToStringPtr("secret_value"),
					Secret: utility.ToBoolPtr(true),
				},
			},
			Platform: utility.ToStringPtr("linux"),
			Secret:   utility.ToStringPtr("secret"),
		}

		res, err := apiPod.ToService()
		require.NoError(t, err)

		p, ok := res.(pod.Pod)
		require.True(t, ok)

		assert.Equal(t, utility.FromStringPtr(apiPod.Secret), p.Secret)
		assert.Equal(t, utility.FromStringPtr(apiPod.Image), p.TaskContainerCreationOpts.Image)
		assert.Equal(t, utility.FromIntPtr(apiPod.Memory), p.TaskContainerCreationOpts.MemoryMB)
		assert.Equal(t, utility.FromIntPtr(apiPod.CPU), p.TaskContainerCreationOpts.CPU)
		assert.Equal(t, utility.FromStringPtr(apiPod.Platform), string(p.TaskContainerCreationOpts.Platform))
		assert.Equal(t, utility.FromStringPtr(apiPod.Secret), p.Secret)
		assert.Equal(t, pod.StatusInitializing, p.Status)
		assert.Len(t, p.TaskContainerCreationOpts.EnvVars, 2)
		assert.Len(t, p.TaskContainerCreationOpts.EnvSecrets, 1)
		assert.Equal(t, "value1", p.TaskContainerCreationOpts.EnvVars["name1"])
		assert.Equal(t, "secret_value", p.TaskContainerCreationOpts.EnvSecrets["secret_name"])
	})
}

func TestAPIPod(t *testing.T) {
	validDBPod := func() pod.Pod {
		return pod.Pod{
			ID:     "id",
			Status: pod.StatusRunning,
			Secret: "secret",
			TaskContainerCreationOpts: pod.TaskContainerCreationOptions{
				Image:    "image",
				MemoryMB: 128,
				CPU:      128,
				EnvVars: map[string]string{
					"var0": "val0",
					"var1": "val1",
				},
				EnvSecrets: map[string]string{
					"secret0": "secret_val0",
					"secret1": "secret_val1",
				},
			},
			Resources: pod.ResourceInfo{
				ID:           "id",
				DefinitionID: "definition_id",
				Cluster:      "cluster",
				SecretIDs:    []string{"secret_id0", "secret_id1"},
			},
		}
	}

	validAPIPod := func() APIPod {
		status := PodStatusRunning
		platform := evergreen.PodPlatformLinux
		return APIPod{
			ID:     utility.ToStringPtr("id"),
			Status: &status,
			Secret: utility.ToStringPtr("secret"),
			TaskContainerCreationOpts: APIPodTaskContainerCreationOptions{
				Image:    utility.ToStringPtr("image"),
				MemoryMB: utility.ToIntPtr(128),
				CPU:      utility.ToIntPtr(128),
				Platform: &platform,
				EnvVars: map[string]string{
					"var0": "val0",
					"var1": "val1",
				},
				EnvSecrets: map[string]string{
					"secret0": "secret_val0",
					"secret1": "secret_val1",
				},
			},
			Resources: APIPodResourceInfo{
				ID:           utility.ToStringPtr("id"),
				DefinitionID: utility.ToStringPtr("definition_id"),
				Cluster:      utility.ToStringPtr("cluster"),
				SecretIDs:    []string{"secret_id0", "secret_id1"},
			},
		}
	}
	t.Run("ToService", func(t *testing.T) {
		t.Run("Succeeds", func(t *testing.T) {
			apiPod := validAPIPod()
			dbPod, err := apiPod.ToService()
			require.NoError(t, err)
			assert.Equal(t, utility.FromStringPtr(apiPod.ID), dbPod.ID)
			assert.Equal(t, pod.StatusRunning, dbPod.Status)
			assert.Equal(t, utility.FromStringPtr(apiPod.Secret), dbPod.Secret)
			assert.Equal(t, utility.FromStringPtr(apiPod.TaskContainerCreationOpts.Image), dbPod.TaskContainerCreationOpts.Image)
			assert.Equal(t, utility.FromIntPtr(apiPod.TaskContainerCreationOpts.MemoryMB), dbPod.TaskContainerCreationOpts.MemoryMB)
			assert.Equal(t, utility.FromIntPtr(apiPod.TaskContainerCreationOpts.CPU), dbPod.TaskContainerCreationOpts.CPU)
			assert.Equal(t, *apiPod.TaskContainerCreationOpts.Platform, dbPod.TaskContainerCreationOpts.Platform)
			require.NotZero(t, dbPod.TaskContainerCreationOpts.EnvVars)
			for k, v := range apiPod.TaskContainerCreationOpts.EnvVars {
				assert.Equal(t, v, dbPod.TaskContainerCreationOpts.EnvVars[k])
			}
			require.NotZero(t, dbPod.TaskContainerCreationOpts.EnvSecrets)
			for k, v := range apiPod.TaskContainerCreationOpts.EnvSecrets {
				assert.Equal(t, v, dbPod.TaskContainerCreationOpts.EnvSecrets[k])
			}
			assert.Equal(t, utility.FromStringPtr(apiPod.Resources.ID), dbPod.Resources.ID)
			assert.Equal(t, utility.FromStringPtr(apiPod.Resources.DefinitionID), dbPod.Resources.DefinitionID)
			assert.Equal(t, utility.FromStringPtr(apiPod.Resources.Cluster), dbPod.Resources.Cluster)
			left, right := utility.StringSliceSymmetricDifference(dbPod.Resources.SecretIDs, apiPod.Resources.SecretIDs)
			assert.Empty(t, left)
			assert.Empty(t, right)
		})
		t.Run("FailsWithInvalidStatus", func(t *testing.T) {
			apiPod := validAPIPod()
			status := APIPodStatus("invalid")
			apiPod.Status = &status
			apiPod.TaskContainerCreationOpts.Platform = nil
			_, err := apiPod.ToService()
			assert.Error(t, err)
		})
		t.Run("FailsWithoutPlatform", func(t *testing.T) {
			apiPod := validAPIPod()
			apiPod.TaskContainerCreationOpts.Platform = nil
			_, err := apiPod.ToService()
			assert.Error(t, err)
		})
	})
	t.Run("BuildFromService", func(t *testing.T) {
		t.Run("Succeeds", func(t *testing.T) {
			dbPod := validDBPod()
			var apiPod APIPod
			require.NoError(t, apiPod.BuildFromService(&dbPod))
			assert.Equal(t, dbPod.ID, utility.FromStringPtr(apiPod.ID))
			require.NotZero(t, apiPod.Status)
			assert.Equal(t, PodStatusRunning, *apiPod.Status)
			assert.Equal(t, dbPod.Secret, utility.FromStringPtr(apiPod.Secret))
			assert.Equal(t, dbPod.TaskContainerCreationOpts.Image, utility.FromStringPtr(apiPod.TaskContainerCreationOpts.Image))
			assert.Equal(t, dbPod.TaskContainerCreationOpts.MemoryMB, utility.FromIntPtr(apiPod.TaskContainerCreationOpts.MemoryMB))
			assert.Equal(t, dbPod.TaskContainerCreationOpts.CPU, utility.FromIntPtr(apiPod.TaskContainerCreationOpts.CPU))
			require.NotZero(t, apiPod.TaskContainerCreationOpts.Platform)
			assert.Equal(t, dbPod.TaskContainerCreationOpts.Platform, *apiPod.TaskContainerCreationOpts.Platform)
			require.NotZero(t, apiPod.TaskContainerCreationOpts.EnvVars)
			for k, v := range dbPod.TaskContainerCreationOpts.EnvVars {
				assert.Equal(t, v, apiPod.TaskContainerCreationOpts.EnvVars[k])
			}
			require.NotZero(t, apiPod.TaskContainerCreationOpts.EnvSecrets)
			for k, v := range dbPod.TaskContainerCreationOpts.EnvSecrets {
				assert.Equal(t, v, apiPod.TaskContainerCreationOpts.EnvSecrets[k])
			}
			assert.Equal(t, dbPod.Resources.ID, utility.FromStringPtr(apiPod.Resources.ID))
			assert.Equal(t, dbPod.Resources.DefinitionID, utility.FromStringPtr(apiPod.Resources.DefinitionID))
			assert.Equal(t, dbPod.Resources.Cluster, utility.FromStringPtr(apiPod.Resources.Cluster))
			left, right := utility.StringSliceSymmetricDifference(dbPod.Resources.SecretIDs, apiPod.Resources.SecretIDs)
			assert.Empty(t, left)
			assert.Empty(t, right)
		})
		t.Run("FailsWithInvalidStatus", func(t *testing.T) {
			dbPod := validDBPod()
			dbPod.Status = "invalid"
			var apiPod APIPod
			assert.Error(t, apiPod.BuildFromService(&dbPod))
		})
	})
}