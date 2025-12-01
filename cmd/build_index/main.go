package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
)

type rawMeal map[string]interface{}

type IndexItem struct {
    IDMeal       string   `json:"idMeal"`
    StrMeal      string   `json:"strMeal"`
    StrArea      string   `json:"strArea"`
    StrCategory  string   `json:"strCategory,omitempty"`
    StrMealThumb string   `json:"strMealThumb,omitempty"`
    Ingredients  []string `json:"ingredients"`
}

func main() {
    inPath := flag.String("file", "all_meals.json", "path to all_meals.json")
    outPath := flag.String("out", "public/meals_index.json", "output index file")
    force := flag.Bool("force", false, "overwrite output if exists")
    flag.Parse()

    if !*force {
        if _, err := os.Stat(*outPath); err == nil {
            fmt.Printf("index already exists at %s — use --force to overwrite\n", *outPath)
            return
        }
    }

    b, err := os.ReadFile(*inPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", *inPath, err)
        os.Exit(2)
    }

    var raw []rawMeal
    if err := json.Unmarshal(b, &raw); err != nil {
        fmt.Fprintf(os.Stderr, "failed to parse JSON: %v\n", err)
        os.Exit(2)
    }

    out := make([]IndexItem, 0, len(raw))

    for _, m := range raw {
        id := toStr(m["idMeal"])
        name := toStr(m["strMeal"])
        area := toStr(m["strArea"])
        cat := toStr(m["strCategory"])
        thumb := toStr(m["strMealThumb"])

        // gather ingredient fields
        var ing []string
        for i := 1; i <= 20; i++ {
            key := fmt.Sprintf("strIngredient%d", i)
            if v, ok := m[key]; ok {
                s := toStr(v)
                if s != "" {
                    ing = append(ing, s)
                }
            }
        }

        item := IndexItem{
            IDMeal:       id,
            StrMeal:      name,
            StrArea:      area,
            StrCategory:  cat,
            StrMealThumb: thumb,
            Ingredients:  ing,
        }
        out = append(out, item)
    }

    ob, err := json.MarshalIndent(out, "", "  ")
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to marshal output: %v\n", err)
        os.Exit(2)
    }

    if err := os.MkdirAll("public", 0755); err != nil {
        fmt.Fprintf(os.Stderr, "failed to create public dir: %v\n", err)
        os.Exit(2)
    }

    if err := os.WriteFile(*outPath, ob, 0644); err != nil {
        fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", *outPath, err)
        os.Exit(2)
    }

    fmt.Printf("wrote %d index items to %s\n", len(out), *outPath)
}

func toStr(v interface{}) string {
    if v == nil {
        return ""
    }
    switch t := v.(type) {
    case string:
        return t
    default:
        return fmt.Sprint(v)
    }
}
