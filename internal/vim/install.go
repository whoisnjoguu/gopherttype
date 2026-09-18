package vim

import (
	"errors"
	"os/exec"
	"runtime"
)

// InstallPlan is a way to install Neovim discovered from whichever package
// manager is present on this machine.
type InstallPlan struct {
	Manager string
	Command []string
	AutoRun bool
}

var errInstallNeedsManualRun = errors.New("vim: this install ne`eds elevated privileges; run it yourself in a terminal")

// installCandidates lists, per OS
var installCandidates = map[string][]struct {
	manager string
	lookup  string
	command []string
	autoRun bool
}{
	"darwin": {
		{"brew", "brew", []string{"brew", "install", "neovim"}, true},
	},
	"linux": {
		{"apt", "apt-get", []string{"sudo", "apt-get", "install", "-y", "neovim"}, false},
		{"dnf", "dnf", []string{"sudo", "dnf", "install", "-y", "neovim"}, false},
		{"pacman", "pacman", []string{"sudo", "pacman", "-S", "--noconfirm", "neovim"}, false},
		{"zypper", "zypper", []string{"sudo", "zypper", "install", "-y", "neovim"}, false},
		{"apk", "apk", []string{"sudo", "apk", "add", "neovim"}, false},
	},
	"windows": {
		{"winget", "winget", []string{"winget", "install", "-e", "--id", "Neovim.Neovim"}, true},
		{"scoop", "scoop", []string{"scoop", "install", "neovim"}, true},
		{"choco", "choco", []string{"choco", "install", "neovim", "-y"}, false},
	},
}

// DetectInstallPlan finds the best available way to install Neovim on this machine
func DetectInstallPlan() (plan InstallPlan, ok bool) {
	for _, c := range installCandidates[runtime.GOOS] {
		if _, err := exec.LookPath(c.lookup); err == nil {
			return InstallPlan{Manager: c.manager, Command: c.command, AutoRun: c.autoRun}, true
		}
	}
	return InstallPlan{}, false
}

// Run executes the install command and blocks until it exits
func (p InstallPlan) Run() error {
	if !p.AutoRun || len(p.Command) == 0 {
		return errInstallNeedsManualRun
	}
	return exec.Command(p.Command[0], p.Command[1:]...).Run()
}
