package main

import (
	"context"
	"encoding/json"
	"os"
)

type JsonExporter struct{}

func (e *JsonExporter) Export(ctx context.Context, path string, countries []Country) error {
	output, err := json.MarshalIndent(countries, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(path+"/data.json", output, 0644)
	return err
}
