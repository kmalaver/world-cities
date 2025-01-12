package main

import (
	"context"
	"encoding/json"
)

type JsonExporter struct{}

func (e *JsonExporter) Ext() string {
	return "json"
}

func (e *JsonExporter) Export(ctx context.Context, countries []Country) ([]byte, error) {
	return json.MarshalIndent(countries, "", "  ")
}
