package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type Config struct {
	TmpFolder string
	TmpPrefix string
	Port      int
}

var DefaultConfig = Config{
	TmpFolder: "/tmp",
	TmpPrefix: "incoming",
	Port:      8080,
}

func main() {
	cfg := DefaultConfig
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	})

	// Handles raw payloads.
	mux.HandleFunc("/raw", func(w http.ResponseWriter, r *http.Request) {
		// Read contents of request into a temporary file to be used as ogr2ogr
		// input.
		inFile, err := ioutil.TempFile(cfg.TmpFolder, cfg.TmpPrefix)
		if err != nil {
			log.Printf("Failed to create inFile")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read input contents")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = inFile.Write(body)
		if err != nil {
			log.Printf("Failed to write contents to inFile")
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			log.Printf("Got contents: %s", body)
		}

		outFile, err := ioutil.TempFile(cfg.TmpFolder, cfg.TmpPrefix)
		if err != nil {
			log.Printf("Failed to create outFile")
			return
		}

		outFileName := outFile.Name()
		err = outFile.Close()
		if err != nil {
			log.Printf("Failed to close outFile")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// This is ghetto but we delete the tmpFile because ogr2ogr doesn't
		// support overwriting files.
		os.Remove(outFileName)

		args := []string{
			"-f",
			"GeoJSON",
			outFile.Name(),
			inFile.Name(),
		}

		err = inFile.Close()
		if err != nil {
			log.Printf("Could not close inFile")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		out, err := exec.Command("ogr2ogr", args...).CombinedOutput()
		if err != nil {
			log.Printf("Error for ogr2ogr: %s, %s", err, out)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "ogr2ogr error: %s", out)
			return
		}

		outFile, err = os.Open(outFileName)
		if err != nil {
			log.Printf("Failed to open output file")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		result, err := ioutil.ReadAll(outFile)
		if err != nil {
			log.Printf("Failed to read resulting json contents")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = os.Remove(outFileName)
		if err != nil {
			log.Printf("Could not remove outFile")
		}

		err = os.Remove(inFile.Name())
		if err != nil {
			log.Printf("Could not remove inFile")
		}

		w.Write(result)
	})

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
	s.ListenAndServe()
}
