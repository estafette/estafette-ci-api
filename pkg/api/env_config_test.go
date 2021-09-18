package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideFromEnv(t *testing.T) {
	t.Run("DoesNotOverrideConfigIfNoEnvironmentVariablesStartWithCapitalizedPrefixUnderscore", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{}
		var config struct {
			MyValue string
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "", config.MyValue)
	})

	t.Run("OverridesConfigFieldIfEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE=bye bye"}
		var config struct {
			MyValue string
		}
		config.MyValue = "hi there"

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "bye bye", config.MyValue)
	})

	t.Run("KeepsConfigFieldIfNoEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUEDOESNOTMATCH=byebye"}
		var config struct {
			MyValue string
		}
		config.MyValue = "hi there"

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "hi there", config.MyValue)
	})

	t.Run("OverridesConfigPointerFieldIfEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE=bye bye"}
		var config struct {
			MyValue *string
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "bye bye", *config.MyValue)
	})

	t.Run("KeepsConfigPointerFieldNilIfNoEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUEDOESNOTMATCH=byebye"}
		var config struct {
			MyValue *string
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Nil(t, config.MyValue)
	})

	t.Run("OverridesConfigArrayFieldIfEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE=bye,now"}
		var config struct {
			MyValue []string
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, 2, len(config.MyValue))
		assert.Equal(t, "bye", config.MyValue[0])
		assert.Equal(t, "now", config.MyValue[1])
	})

	t.Run("KeepsConfigFieldIfNoEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUEDOESNOTMATCH=byebye"}
		var config struct {
			MyValue []string
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(config.MyValue))
	})

	t.Run("OverridesNestedConfigFieldIfEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE_INNERVALUE=bye bye"}
		var config struct {
			MyValue struct {
				InnerValue string
			}
		}
		config.MyValue.InnerValue = "hi there"

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "bye bye", config.MyValue.InnerValue)
	})

	t.Run("KeepsNestedConfigFieldIfNoEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE_INNERVALUEDOESNOTMATCH=byebye"}
		var config struct {
			MyValue struct {
				InnerValue string
			}
		}
		config.MyValue.InnerValue = "hi there"

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "hi there", config.MyValue.InnerValue)
	})

	t.Run("OverridesNestedConfigPointerFieldIfEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE_INNERVALUE=bye bye"}
		var config struct {
			MyValue *struct {
				InnerValue string
			}
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "bye bye", config.MyValue.InnerValue)
	})

	t.Run("KeepsNestedConfigPointerFieldIfNoEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExistsButOneWithNestedPrefixDoes", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_MYVALUE_INNERVALUEDOESNOTMATCH=byebye"}
		var config struct {
			MyValue *struct {
				InnerValue string
			}
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Equal(t, "", config.MyValue.InnerValue)
	})

	t.Run("KeepsNestedConfigPointerFieldIfNoEnvironmentVariableMatchingCapitalizedPrefixUnderscoreFieldNameExists", func(t *testing.T) {

		prefix := "prefix"
		environmentVariables := []string{"PREFIX_INNERVALUEDOESNOTMATCH=byebye"}
		var config struct {
			MyValue *struct {
				InnerValue string
			}
		}

		// act
		err := OverrideFromEnv(&config, prefix, environmentVariables)

		assert.Nil(t, err)
		assert.Nil(t, config.MyValue)
	})
}
