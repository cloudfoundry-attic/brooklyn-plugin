package io

import (
	//"fmt"
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"path/filepath"
	. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/generic"
	"github.com/cloudfoundry/cli/cf/errors"
	"os"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"io"
)



func ReadYAMLFile(path string) generic.Map {
	file, err := os.Open(filepath.Clean(path))
	assert.ErrorIsNil(err)
	defer file.Close()

	yamlMap, err := parse(file)
	assert.ErrorIsNil(err)
	return yamlMap
}

func parse(file io.Reader) (yamlMap generic.Map, err error) {
	decoder := candiedyaml.NewDecoder(file)
	yamlMap = generic.NewMap()
	err = decoder.Decode(yamlMap)
	
	assert.ErrorIsNil(err)

	if !generic.IsMappable(yamlMap) {
		err = errors.New(T("Invalid. Expected a map"))
		return
	}

	return
}


func WriteYAMLFile(yamlMap generic.Map, path string) {

	fileToWrite, err := os.Create(path)
	assert.ErrorIsNil(err)

	encoder := candiedyaml.NewEncoder(fileToWrite)
	err = encoder.Encode(yamlMap)

	assert.ErrorIsNil(err)

	return
}