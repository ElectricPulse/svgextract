package main

import (
	"fmt"
  "strings"
  "bufio"
	"os"
	"path/filepath"
  "github.com/galihrivanto/go-inkscape"
  "github.com/goccy/go-yaml"
  "errors"
)

type Config struct {
	input string
  unlinked_input string
	output_folder string
}

func save_manifest(proxy *inkscape.Proxy, manifest Manifest, config Config) error {
  // Using json because Typescript can natively infer types from it
	path := filepath.Join(config.output_folder, "manifest.json")

  data, err := yaml.MarshalWithOptions(&manifest, yaml.JSON())

  if err != nil {
    return err
  }

  err = os.WriteFile(path, data, 0644)

  return err
}

func safely_delete(path string) (bool, error) {
  reader := bufio.NewReader(os.Stdin)
  fmt.Printf("Folder %q exists and contains files. Delete its contents? (y/N): ", path)
  answer, err := reader.ReadString('\n')

  if err != nil {
    return false, err
  }
  answer = strings.TrimSpace(strings.ToLower(answer))

  if answer != "y" && answer != "yes" {
    return true, nil
  }

  // User confirmed: delete contents
  fmt.Println("Deleting contents...")

  entries, err := os.ReadDir(path)

  if err != nil {
    return false, err
  }

  for _, entry := range entries {
    entryPath := filepath.Join(path, entry.Name())
    if err := os.RemoveAll(entryPath); err != nil {
      return false, err
    }
  }

  return false, nil
}

func program(config Config) error {
  proxy, err := create_proxy(config.input)
  defer proxy.Close()

  unlinked_input, err := os.CreateTemp("", "extractor-temp-*.svg")

  if err != nil {
    return err
  }

  defer unlinked_input.Close()

  unlinked_svg := unlinked_input.Name()

  args := []string{
    "select-all", "clone-unlink-recursively", "select-clear",
    inkscape.ExportFileName(unlinked_svg), inkscape.ExportDo(),
  }

  _, err = proxy.RawCommandsContext(proxy_context, args...)

  if err != nil {
    return err
  }

  config.unlinked_input = unlinked_svg

  info, err := os.Stat(config.output_folder)

  if err != nil {
    if ! os.IsNotExist(err) {
      return err
    }

    if err := os.Mkdir(config.output_folder, 0755); err != nil {
      return err
    }
  } else if info.IsDir() {
    declined, err := safely_delete(config.output_folder)

    if err != nil {
      return err
    }

    if declined {
      return errors.New("Change output_folder in the config or delete it's contents")
    }
  } else {
    return fmt.Errorf("File %q exists but is not a folder", config.output_folder)
  }

	manifest, err := process(proxy, config)

	if err != nil {
		return err
	}

  if err := save_manifest(proxy, manifest, config); err != nil {
    return err
  }

  return nil
}

func main() {
  if len(os.Args) != 3 {
    fmt.Fprintf(os.Stderr, "Invalid arguments: provide <input-inkscape.svg> and <output-dir>\n")
    return
  }

  err := program(Config {
    input: os.Args[1],
    output_folder: os.Args[2],
  })

  if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
  }
}
