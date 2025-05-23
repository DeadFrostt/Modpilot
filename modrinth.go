package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path"
)

type Version struct {
    ID           string   `json:"id"`
    GameVersions []string `json:"game_versions"`
    Loaders      []string `json:"loaders"`
    Files []struct {
        URL      string `json:"url"`
        Filename string `json:"filename"`
    } `json:"files"`
}

// FetchLatestVersion queries Modrinth for the newest version matching MC+loader
func FetchLatestVersion(slug, mcVersion, loader string) (*Version, error) {
    url := fmt.Sprintf(
        "https://api.modrinth.com/v2/project/%s/version?loaders=%s&game_versions=%s",
        slug, loader, mcVersion,
    )
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var versions []Version
    if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
        return nil, err
    }
    if len(versions) == 0 {
        return nil, fmt.Errorf("no versions found for %s", slug)
    }
    // find first version whose game_versions includes mcVersion and loaders includes loader
    for _, v := range versions {
        okGV := false
        for _, gv := range v.GameVersions {
            if gv == mcVersion {
                okGV = true
                break
            }
        }
        if !okGV {
            continue
        }
        for _, ld := range v.Loaders {
            if ld == loader {
                return &v, nil
            }
        }
    }
    return nil, fmt.Errorf("no compatible version found for %s (MC %s, loader %s)", slug, mcVersion, loader)
}

// DownloadFile streams the URL to destDir/<filename>
func DownloadFile(url, destDir string) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if err := os.MkdirAll(destDir, 0755); err != nil {
        return "", err
    }
    fname := path.Base(url)
    outPath := path.Join(destDir, fname)
    out, err := os.Create(outPath)
    if err != nil {
        return "", err
    }
    defer out.Close()

    if _, err := io.Copy(out, resp.Body); err != nil {
        return "", err
    }
    return outPath, nil
}
