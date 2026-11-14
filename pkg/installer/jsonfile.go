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

// modeOrDefault devuelve el modo actual de path para conservarlo al
// reescribirlo, o def si no existe. Otro error impide la escritura: sin conocer
// el modo no se puede garantizar que no se amplíe (HCD-032).
func modeOrDefault(path string, def os.FileMode) (os.FileMode, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return def, nil
	}
	if err != nil {
		return 0, fmt.Errorf("leer permisos de %s: %w", path, err)
	}
	return info.Mode().Perm(), nil
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

// linkNotFileError indica que una configuración de Pi es un enlace roto, un
// bucle o un enlace a algo que no es un archivo. No envuelve fs.ErrNotExist ni
// ENOTDIR a propósito: nunca debe confundirse con un archivo ausente (HCD-036).
type linkNotFileError struct {
	path   string
	detail string
}

func (e *linkNotFileError) Error() string {
	return fmt.Sprintf("%s: el enlace no apunta a un archivo (%s); no se modifica", e.path, e.detail)
}

// configTarget devuelve la ruta donde leer y escribir una configuración de Pi
// (mcp.json, settings.json, AGENTS.md o pi-cc-extensions.json). Si path es un
// enlace simbólico devuelve su destino final, para escribir a través del enlace
// sin sustituirlo; si el enlace no lleva a un archivo, devuelve
// *linkNotFileError. Si path no es un enlace o no existe, devuelve path tal
// cual y el llamador sigue como hasta ahora (HCD-036, H-02).
func configTarget(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&fs.ModeSymlink == 0 {
		return path, nil
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", &linkNotFileError{path: path, detail: "destino inexistente o bucle de enlaces"}
	}
	st, err := os.Stat(real)
	if err != nil {
		return "", &linkNotFileError{path: path, detail: err.Error()}
	}
	if !st.Mode().IsRegular() {
		return "", &linkNotFileError{path: path, detail: "apunta a " + real + ", que no es un archivo"}
	}
	return real, nil
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
