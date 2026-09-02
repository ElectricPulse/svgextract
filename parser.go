package main

import (
  "fmt"
  "os"
  "sort"
  "path/filepath"
  "strings"
  "encoding/xml"
  "github.com/galihrivanto/go-inkscape"
)

type Manifest_element struct {
  Position Position
  Dimensions Dimensions
  Path string
  Id string
}

type Manifest_node interface {
  Manifest_element | Manifest
}

// Contains manifest_node
type Manifest map[string] any


type Element struct {
  XMLName xml.Name
  Label string `xml:"label,attr"`
  Id string `xml:"id,attr"`
  Transform string `xml:"transform,attr"`
  X float64 `xml:"x,attr"`
  Y float64 `xml:"y,attr"`
  Children []Element `xml:",any"`
  Reference string `xml:"href,attr"`
  Attrs []xml.Attr `xml:",any,attr"`
}

// Commands
const (
  save_cmd = "save"
  save_children_cmd = "save-children"
  center_position_cmd = "center-position"
  ignore_grouping_cmd = "ignore-grouping"
)

func get_translation(x float64, y float64) []float64 {
  return []float64{1, 0, x, 0, 1, y, 0, 0, 1}
}

const command_prefix = "#"

func calculate_position(corner_position Position, dimensions Dimensions, center bool) Position {
  if center {
    return Position {
      X: corner_position.X + dimensions.Width/2,
      Y: corner_position.Y + dimensions.Height/2,
    }
  }

  return corner_position
}

func create_manifest(proxy *inkscape.Proxy, element Element, path string, center_position bool) (any, error) {
  fmt.Printf("Creating manifest for %q\n", filepath.Join(path))

  var err error

  corner_position, err := inkscape_get_position(proxy, element.Id)

  if err != nil {
    return Manifest_element{}, err
  }

  dimensions, err := inkscape_get_dimensions(proxy, element.Id)

  if err != nil {
    return Manifest_element{}, err
  }

  return Manifest_element {
    Position: calculate_position(corner_position, dimensions, center_position),
    Dimensions: dimensions,
    Path: path,
    Id: element.Id,
  }, nil
}

func merge_manifests(a Manifest, b Manifest) Manifest {
  for key, value := range b {
    a[key] = value
  }

  return a
}

var symbol_stage bool
var symbols Manifest

func change_position(original_manifest any, original_label string, position Position) Manifest {
  manifest := make(Manifest)

  if element, ok := original_manifest.(Manifest_element); ok {
    manifest[original_label] = Manifest_element {
      Position: position,
      Dimensions: element.Dimensions,
      Path: element.Path,
      Id: element.Id,
    }

    return manifest
  }

  if original_manifest, ok := original_manifest.(Manifest); ok {
    children_manifest := make(Manifest)

    for label := range(original_manifest) {
      if label == "Id" {
        continue
      }

      child_manifest := change_position(original_manifest[label], label, position)
      children_manifest = merge_manifests(children_manifest, child_manifest)
    }

    manifest[original_label] = children_manifest

    return manifest
  }

  panic("Internal error")
}

func get_symbol_manifest(symbols Manifest, id string, symbol_label string, position Position, center_position bool) Manifest {
  for label := range(symbols) {
    symbol := symbols[label]

    var symbol_id string
    var dimensions Dimensions

    if child_manifest, ok := symbol.(Manifest); ok {
      symbol_id = child_manifest["Id"].(string)
    } else if element, ok := symbol.(Manifest_element); ok {
      dimensions = element.Dimensions
      symbol_id = element.Id
    } else {
      continue
    }

    if id == symbol_id {
      fmt.Printf("Applying symbol of id %s for object %s\n", symbol_label, id)
      return change_position(symbol, symbol_label, calculate_position(position, dimensions, center_position))
    }
  }

  return make(Manifest)
}

type Command struct {
  save bool
  center_position bool
  save_children bool
  ignore_grouping bool
}

func get_command(label string) Command {
  return Command {
    save: strings.Contains(label, save_cmd),
    save_children: strings.Contains(label, save_children_cmd),
    ignore_grouping: strings.Contains(label, ignore_grouping_cmd),
    center_position: strings.Contains(label, center_position_cmd),
  }
}

func get_label(label string) string {
  if idx := strings.Index(label, command_prefix); idx >= 0 {
    return label[:idx]
  }

  return label
}

func get_name(element Element) (string, Command) {
  if element.Label == "" {
    return element.Id, Command {}
  }

  return get_label(element.Label), get_command(element.Label)
}

func process_element(config Config, proxy *inkscape.Proxy, element Element, cmd Command, path string) (Manifest, error) {
  manifest := make(Manifest)
  var direct_cmd Command

  element.Label, direct_cmd = get_name(element)

  // Is a symbol
  if len(element.Reference) > 1 && len(symbols) != 0 {
    // Leave out the hashtag
    reference := element.Reference[1:]
    position, err := inkscape_get_position(proxy, element.Id)

    if err != nil {
      return manifest, err
    }

    return get_symbol_manifest(symbols, reference, element.Label, position, direct_cmd.center_position || cmd.center_position), nil
  }

  if direct_cmd.save_children {
    children_manifest := make(Manifest)

    sort.Slice(element.Children, func(i, j int) bool {
      name1, _ := get_name(element.Children[i])
      name2, _ := get_name(element.Children[j])
      return name1 < name2
    })

    new_path := filepath.Join(path, element.Label)

    child_cmd := Command {
      save: true,
      center_position: direct_cmd.center_position,
    }

    for _, child := range(element.Children) {
      child_manifest, err := process_element(config, proxy, child, child_cmd, new_path)

      if err != nil {
        return Manifest{}, err
      }

      children_manifest = merge_manifests(children_manifest, child_manifest)
    }

    if symbol_stage {
      children_manifest["Id"] = element.Id
    }

    manifest[element.Label] = children_manifest

    return manifest, nil
  }

  if (cmd.save || direct_cmd.save) && !direct_cmd.ignore_grouping {
    dir := filepath.Join(config.output_folder, path)

    if err := os.MkdirAll(dir, 0755); err != nil {
      fmt.Printf("failed to create directory %q: %v\n", dir, err)
      return manifest, err
    }

    manifest_path := filepath.Join(path, element.Label + ".svg")
    svg := filepath.Join(config.output_folder, manifest_path)

    err := inkscape_save_svg(proxy, config, element.Id, svg)

    if err != nil {
      return manifest, err
    }

    fmt.Printf("Saved %s\n", svg)

    element_manifest, err := create_manifest(proxy, element, manifest_path, cmd.center_position || direct_cmd.center_position)
    manifest[element.Label] = element_manifest

    return manifest, err
  }

  // Recurse
  for _, child := range(element.Children) {
    child_manifest, err := process_element(config, proxy, child, direct_cmd, path)

    if err != nil {
      return nil, err
    }

    manifest = merge_manifests(manifest, child_manifest)
  }

  return manifest, nil
}

func process(proxy *inkscape.Proxy, config Config) (Manifest, error) {
  file, err := os.ReadFile(config.input)

  if err != nil {
    return nil, err
  }

  var root Element

  err = xml.Unmarshal(file, &root)

  if err != nil {
    return nil, err
  }

	var main_layer Element
  var symbol_layer Element

  for _, child := range(root.Children) {
    name, _ := get_name(child)

    if name == "symbols" {
      symbol_layer = child
      continue
    }

    if name == "main" {
      main_layer = child
      continue
    }
  }

  if symbol_layer.XMLName.Local != "" {
    symbol_stage = true
    symbols_manifest, err := process_element(config, proxy, symbol_layer, Command{}, "")

    if err != nil {
      return nil, err
    }

    if symbols_manifest["symbols"] != nil {
      symbols = symbols_manifest["symbols"].(Manifest)
    }

    symbol_stage = false
  }

  manifest, err := process_element(config, proxy, main_layer, Command{}, "")

  if err != nil {
    return nil, err
  }

  return manifest, err
}
