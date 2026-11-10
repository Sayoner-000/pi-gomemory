package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// readJSONObject lee un objeto JSON conservando cada valor sin interpretar.
// Un archivo inexistente o vacío devuelve un mapa vacío; un JSON inválido
// devuelve error para no sobrescribir configuración que no se pudo leer.
func readJSONObject(path string) (map[string]json.RawMessage, error) {
	obj := map[string]json.RawMessage{}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(data) == 0) {
		return obj, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leer %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("%s no es JSON válido; no se modifica: %w", path, err)
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	return obj, nil
}

// writeJSONObject serializa el objeto con sangría de dos espacios.
func writeJSONObject(path string, obj any) error {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	return nil
}

// setJSONField asigna obj[key] = value, serializado.
func setJSONField(obj map[string]json.RawMessage, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("serializar %s: %w", key, err)
	}
	obj[key] = raw
	return nil
}
