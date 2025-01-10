package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	assemblyai "github.com/AssemblyAI/assemblyai-go-sdk"
	_ "github.com/joho/godotenv/autoload"
)

const (
	banner = `
 .d8888b.   .d88888b.  888     888 8888888 8888888b.  888     888 d8b                           
d88P  Y88b d88P" "Y88b 888     888   888   888  "Y88b 888     888 Y8P                           
888    888 888     888 888     888   888   888    888 888     888                               
888        888     888 Y88b   d88P   888   888    888 Y88b   d88P 888 .d8888b   .d88b.  888d888 
888        888     888  Y88b d88P    888   888    888  Y88b d88P  888 88K      d88""88b 888P"   
888    888 888     888   Y88o88P     888   888    888   Y88o88P   888 "Y8888b. 888  888 888     
Y88b  d88P Y88b. .d88P    Y888P      888   888  .d88P    Y888P    888      X88 Y88..88P 888     
 "Y8888P"   "Y88888P"      Y8P     8888888 8888888P"      Y8P     888  88888P'  "Y88P"  888     
                                                                                                
                                                                                                                                                                                                
`
)

var (
	client *assemblyai.Client = assemblyai.NewClient(os.Getenv("ASSEMBLY_AI_API_KEY"))
)

func main() {
	fmt.Print(banner)

	var shouldSeed bool
	flag.BoolVar(&shouldSeed, "seed", false, "Whether covid data should be seed into the database from the file datasets/covid.csv")
	flag.Parse()

	pythonProcessIntentAndParameters, err := NewPython("scripts/process_intent_and_parameters.py")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := pythonProcessIntentAndParameters.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	pythonProcessCustomQuery, err := NewPython("scripts/process_custom_query.py")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := pythonProcessCustomQuery.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	appdb, err := NewDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer appdb.Close()

	if shouldSeed {
		covidData, err := seedCovidData()
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := appdb.InsertCovidData(covidData); err != nil {
			fmt.Println(err)
			return
		}
	}

	dir, err := os.MkdirTemp("", "covid-visor")
	if err != nil {
		fmt.Println("unable to create temporary directory,", err)
		return
	}
	defer os.RemoveAll(dir)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
scanner:
	for {
		fmt.Println("PRESS ENTER TO START RECORDING OR PRESS CTRL-C TO TERMINATE THE PROGRAM")
		fmt.Scanln()

		select {
		case <-shutdownCh:
			break scanner
		default:
		}

		filename, err := recordAudio(dir)
		if err != nil {
			fmt.Println(err)
			continue
		}

		text, err := transcribeAudio(filename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		intentAndParameters, err := pythonProcessIntentAndParameters.Input(text)
		if err != nil {
			fmt.Println(err)
			continue
		}

		result, isCustom, err := appdb.ProcessQuery(intentAndParameters)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if isCustom {
			result, err = pythonProcessCustomQuery.Input(text)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}

		if err := playText(result); err != nil {
			fmt.Println(err)
		}
	}
}

func recordAudio(dir string) (string, error) {
	file, err := os.CreateTemp(dir, "*.wav")
	if err != nil {
		return "", fmt.Errorf("unable to create a temporary file, %w", err)
	}
	_ = file.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		fmt.Println("TO STOP RECORDING, PRESS ENTER")
		fmt.Scanln()
		cancel()
	}()

	cmd := exec.CommandContext(ctx, "arecord", "-f", "cd", file.Name())
	if err := cmd.Run(); err != nil && ctx.Err() == nil {
		return "", fmt.Errorf("unable to initiate recording, %w", err)
	}

	return file.Name(), nil
}

func transcribeAudio(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("unable to open audio file, %w", err)
	}
	defer file.Close()

	transcript, err := client.Transcripts.TranscribeFromReader(context.Background(), file, nil)
	if err != nil {
		return "", fmt.Errorf("unable to transcribe audio file, %w", err)
	}
	return *transcript.Text, nil
}

func playText(text string) error {
	cmd := exec.Command("espeak", fmt.Sprintf(`"%v"`, text))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to play text, %w", err)
	}
	return nil
}
