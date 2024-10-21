package main

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func ParseConfig() (*Config, error) {
	file, err := os.Open("config.yml");
	if err!=nil{
		return nil,err;
	}
	// Read the file content
	data, err := io.ReadAll(file) // Reads the whole file at once
	if err != nil {
		return nil, err
	}
	config := Config{}
	err = yaml.Unmarshal(data,&config)
	if err!=nil{
		return nil,err;
	}
	return &config,nil;
}