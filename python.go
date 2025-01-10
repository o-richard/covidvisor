package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Python struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func NewPython(program string) (*Python, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("unable to get current working directory, %w", err)
	}
	cmd := exec.Command("python", program)
	cmd.Env = append(cmd.Env, "VIRTUAL_ENV_PROMPT=venv", fmt.Sprintf("VIRTUAL_ENV=%v/venv", dir), fmt.Sprintf("PATH=%v/venv/bin:$PATH", dir))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to create stdin pipe, %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to create stdout pipe, %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start python subprocess, %w", err)
	}
	return &Python{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

func (p *Python) Input(s string) (string, error) {
	if _, err := p.stdin.Write(append([]byte(s), '\n')); err != nil {
		return "", fmt.Errorf("unable to process stdin, %w", err)
	}
	output := make([]byte, 1024)
	n, err := p.stdout.Read(output)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("unable to read from stdout, %w", err)
	}
	return string(output[:n]), nil
}

func (p *Python) Close() error {
	if _, err := p.stdin.Write([]byte("q\n")); err != nil {
		return fmt.Errorf("unable to send close signal to stdin, %w", err)
	}
	if err := p.stdin.Close(); err != nil {
		return fmt.Errorf("unable to close stdin, %w", err)
	}
	if err := p.cmd.Wait(); err != nil {
		return fmt.Errorf("unable to wait for python subprocess to exit, %w", err)
	}
	return nil
}
