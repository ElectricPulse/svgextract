package main

import (
  "fmt"
  "os"
  "time"
  "os/exec"
  "errors"
  "strings"
  "strconv"
  "context"
  "path/filepath"
  "github.com/galihrivanto/go-inkscape"
)

type Position struct {
  X float64
  Y float64
}

func create_proxy(input string) (*inkscape.Proxy, error) {
    proxy := inkscape.NewProxy()
    err := proxy.Run()

    if err != nil {
      return nil, err
    }

    args := []string{inkscape.FileOpen(input)}
    _, err = proxy.RawCommandsContext(proxy_context, args...)

    if err != nil {
      return nil, err
    }

    return proxy, err
}

var proxy_context = context.Background()

func _inkscape_query(proxy *inkscape.Proxy, id string, query string) (float64, error) {
    args := []string{"select-by-id:" + id, query, "unselect-by-id:" + id}
    terminal, err := proxy.RawCommandsContext(proxy_context, args...)

    if err != nil {
        return 0, err
    }

    echo := len(strings.Join(args, ";"))

    if len(terminal) < echo {
      return 0, fmt.Errorf("Returned invalid response %q", terminal)
    }

    // Indexing at echo not at echo - 1 because there is an extra : after it
    out := string(terminal[echo:])

    // Apparently inkscape returns nothing if it's zero
    if out == "" {
      return 0, errors.New("Invalid empty response")
    }

    value, err := strconv.ParseFloat(out, 64)

    if err != nil {
      return 0, err
    }

    return value, nil
}

func inkscape_query(proxy *inkscape.Proxy, id string, query string) (float64, error) {
  var value float64
  var err error

  for i := 0; i < 5; i++ {
    value, err = _inkscape_query(proxy, id, query)

    if err == nil {
      break
    }

    fmt.Fprintf(os.Stderr, "Error while running query: %v, Retrying\n", err)

    time.Sleep(time.Second)
  }

  return value, err
}

func inkscape_get_position(proxy *inkscape.Proxy, id string) (Position, error) {
    x, err := inkscape_query(proxy, id, "query-x")

    if err != nil {
      return Position{}, err
    }

    y, err := inkscape_query(proxy, id, "query-y")

    if err != nil {
      return Position{}, err
    }

    return Position {
      X: x,
      Y: y,
    }, nil
}

type Dimensions struct {
  Width float64
  Height float64
}

func inkscape_get_dimensions(proxy *inkscape.Proxy, id string) (Dimensions, error) {
    width, err := inkscape_query(proxy, id, "query-width")

    if err != nil {
      return Dimensions{}, err
    }

    height, err := inkscape_query(proxy, id, "query-height")

    if err != nil {
      return Dimensions{}, err
    }

    return Dimensions {
      Width: width,
      Height: height,
    }, nil
}

func inkscape_save_svg(proxy *inkscape.Proxy, config Config, id string, svg_filepath string) error {
  args := []string{
    "file-close", "file-open:" + config.unlinked_input,
    "export-type:svg", "export-plain-svg", "export-id-only:true", "export-id:" + id,
    inkscape.ExportFileName(svg_filepath), inkscape.ExportDo(),
    "file-close", "file-open:" + config.input,
  }
  _, err := proxy.RawCommandsContext(proxy_context, args...)

  if err != nil {
    return err
  }

  path, err := os.Executable()

  if err != nil {
    return err
  }

  cmd := exec.Command("node", filepath.Join(filepath.Dir(path), "optimize.js"), svg_filepath)

  output, err := cmd.CombinedOutput()

  if err != nil {
    fmt.Fprintf(os.Stderr, "SVGO Error: %v, Output: %s", err, string(output))
    return err
  }

  return nil
}
