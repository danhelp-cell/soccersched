package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"pitchboard/internal/server"
)

var version = "1.0.0"

func main() {
	portFlag := flag.Int("port", 0, "port to listen on (0 picks the saved port or 47821)")
	dataFlag := flag.String("data", "", "folder for the schedule file")
	noBrowser := flag.Bool("no-browser", false, "do not open a browser")
	flag.Parse()

	dataDir := resolveData(*dataFlag)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		showError("Pitchboard", "Could not create the data folder:\n"+dataDir+"\n"+err.Error())
		os.Exit(1)
	}
	setupLog(dataDir)

	port := *portFlag
	if port == 0 {
		port = readPort(dataDir)
	}
	if port == 0 {
		port = 47821
	}

	if !*noBrowser && healthOK(port) {
		_ = openBrowser(fmt.Sprintf("http://127.0.0.1:%d/", port))
		return
	}

	app, err := server.New(dataDir, version)
	if err != nil {
		showError("Pitchboard", err.Error())
		os.Exit(1)
	}
	app.OnShutdown(func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	})

	ln, actual, err := listen(port)
	if err != nil {
		showError("Pitchboard", "Could not start on this computer.\n"+err.Error())
		os.Exit(1)
	}
	writePort(dataDir, actual)
	url := fmt.Sprintf("http://127.0.0.1:%d/", actual)
	log.Printf("Pitchboard %s at %s (data %s)", version, url, dataDir)
	if !*noBrowser {
		go func() {
			time.Sleep(250 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				showError("Pitchboard", "Open this address in your browser:\n"+url)
			}
		}()
	}
	if err := http.Serve(ln, app.Handler()); err != nil {
		showError("Pitchboard", err.Error())
		os.Exit(1)
	}
}

func resolveData(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := strings.TrimSpace(os.Getenv("PITCHBOARD_HOME")); env != "" {
		return env
	}
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		if !strings.Contains(exe, "go-build") && !strings.Contains(filepath.ToSlash(exe), "/Temp/") {
			return filepath.Join(filepath.Dir(exe), "PitchboardData")
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return filepath.Join(cwd, "PitchboardData")
}

func listen(preferred int) (net.Listener, int, error) {
	for port := preferred; port < preferred+20; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return ln, port, nil
		}
	}
	return nil, 0, fmt.Errorf("ports %d-%d are all in use", preferred, preferred+19)
}

func healthOK(port int) bool {
	client := http.Client{Timeout: 400 * time.Millisecond}
	res, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
	if err != nil {
		return false
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	return res.StatusCode == 200 && strings.Contains(string(body), "pitchboard")
}

func readPort(dir string) int {
	raw, err := os.ReadFile(filepath.Join(dir, "port.txt"))
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n < 1 || n > 65535 {
		return 0
	}
	return n
}

func writePort(dir string, port int) {
	_ = os.WriteFile(filepath.Join(dir, "port.txt"), []byte(strconv.Itoa(port)+"\n"), 0o644)
}

func setupLog(dir string) {
	f, err := os.OpenFile(filepath.Join(dir, "pitchboard.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.SetFlags(log.LstdFlags)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
