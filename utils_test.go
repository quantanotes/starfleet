package main

import "testing"

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("config.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("config: %v \n", config)
}

func TestIsJsonPath(t *testing.T) {
	data := map[string]any{
		"foo": map[string]any{
			"bar": true,
		},
		"baz": map[string]any{
			"foobaz": false,
		},
	}
	t.Logf("data: %v\n", data)
	result := IsJsonPath(data, []string{"foo", "bar"})
	t.Logf("foo, baz is: %v\n", result)
	if !result {
		t.Fail()
	}
}
