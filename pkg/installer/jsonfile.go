package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// writeJSONObject serializa el objeto con sangría de dos espacios y lo escribe
// de forma atómica (temporal + rename): un corte nunca deja el archivo truncado.
func writeJSONObject(path string, obj any) error {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar %s: %w", path, err)
	}
	// Keep existing sharing choices; newly created configuration is private.
	mode := os.FileMode(0600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read permissions for %s: %w", path, err)
	}
	if err := writeFileAtomic(path, append(data, '\n'), mode); err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	return os.Rename(name, path)
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
