package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/BRO3886/healthsync/internal/skills"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var skillsAgentFlag string

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage AI agent skills for healthsync",
	Long: `Install, uninstall, and check the status of the healthsync agent skill.

The healthsync skill teaches AI coding agents (Claude Code, Codex CLI, etc.)
how to query Apple Health data effectively. It includes command references,
database schema docs, and SQL query examples.`,
}

func init() {
	rootCmd.AddCommand(skillsCmd)
}

// --- skills install ---

var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install healthsync skill for AI agents",
	Long: `Installs the healthsync agent skill to the selected AI agent's skill directory.

Supported agents:
  claude    → ~/.claude/skills/healthsync/   (Claude Code, Copilot, Cursor, OpenCode, Augment)
  codex     → ~/.agents/skills/healthsync/   (Codex CLI, Copilot, Windsurf, OpenCode, Augment)
  openclaw  → ~/.openclaw/skills/healthsync/ (OpenClaw)

Without --agent, shows an interactive picker to select which agents to install for.`,
	RunE: runSkillsInstall,
}

func init() {
	skillsInstallCmd.Flags().StringVar(&skillsAgentFlag, "agent", "", "Agent target: claude, codex, openclaw, or all")
	skillsCmd.AddCommand(skillsInstallCmd)
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	allTargets := skills.DefaultTargets(homeDir)
	targets, err := resolveTargets(allTargets, skillsAgentFlag, homeDir, "install")
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil // user cancelled
	}

	return installToTargets(EmbeddedSkills, targets, homeDir)
}

func installToTargets(embeddedFS fs.FS, targets []skills.AgentTarget, homeDir string) error {
	green := color.New(color.FgGreen, color.Bold)

	for _, t := range targets {
		written, err := skills.Install(embeddedFS, t, Version)
		if err != nil {
			return fmt.Errorf("failed to install for %s: %w", t.Name, err)
		}

		green.Print("✓ ")
		fmt.Printf("Installed healthsync skill to %s\n", skills.DisplayPath(skills.SkillDir(t), homeDir))
		fmt.Printf("  Files: %s\n", strings.Join(written, ", "))
	}

	fmt.Println("\nThe skill will be available in your next session.")
	return nil
}

// --- skills uninstall ---

var skillsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall healthsync skill from AI agents",
	Long:  `Removes the healthsync agent skill from the selected AI agent's skill directory.`,
	RunE:  runSkillsUninstall,
}

func init() {
	skillsUninstallCmd.Flags().StringVar(&skillsAgentFlag, "agent", "", "Agent target: claude, codex, openclaw, or all")
	skillsCmd.AddCommand(skillsUninstallCmd)
}

func runSkillsUninstall(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	allTargets := skills.DefaultTargets(homeDir)
	targets, err := resolveTargets(allTargets, skillsAgentFlag, homeDir, "uninstall")
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil // user cancelled
	}

	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)

	for _, t := range targets {
		removed, err := skills.Uninstall(t)
		if err != nil {
			return fmt.Errorf("failed to uninstall from %s: %w", t.Name, err)
		}
		if removed {
			green.Print("✓ ")
			fmt.Printf("Removed healthsync skill from %s\n", skills.DisplayPath(skills.SkillDir(t), homeDir))
		} else {
			yellow.Print("- ")
			fmt.Printf("Not installed at %s\n", skills.DisplayPath(skills.SkillDir(t), homeDir))
		}
	}

	return nil
}

// --- skills status ---

var skillsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show skill installation status",
	RunE:  runSkillsStatus,
}

func init() {
	skillsCmd.AddCommand(skillsStatusCmd)
}

func runSkillsStatus(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	allTargets := skills.DefaultTargets(homeDir)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	fmt.Printf("healthsync skill (binary %s):\n", Version)
	for _, t := range allTargets {
		displayDir := skills.DisplayPath(skills.SkillDir(t), homeDir)
		if !skills.IsInstalled(t) {
			red.Printf("  ✗ ")
			fmt.Printf("%-12s %s (not installed)\n", t.Name, displayDir)
			continue
		}
		installed := skills.InstalledVersion(t)
		if installed == "" {
			yellow.Printf("  ? ")
			fmt.Printf("%-12s %s (installed, unknown version)\n", t.Name, displayDir)
		} else if installed != Version {
			yellow.Printf("  ⚠ ")
			fmt.Printf("%-12s %s (installed %s, outdated)\n", t.Name, displayDir, installed)
		} else {
			green.Printf("  ✓ ")
			fmt.Printf("%-12s %s (installed %s)\n", t.Name, displayDir, installed)
		}
	}

	return nil
}

// --- shared helpers ---

// resolveTargets determines which agent targets to operate on.
func resolveTargets(allTargets []skills.AgentTarget, agentFlag, homeDir, action string) ([]skills.AgentTarget, error) {
	if agentFlag != "" {
		return resolveAgentFlag(allTargets, agentFlag)
	}

	if !isInteractive() {
		detected := skills.DetectAgents(allTargets)
		if len(detected) == 0 {
			return allTargets[:1], nil
		}
		return detected, nil
	}

	return runAgentPicker(allTargets, homeDir, action)
}

func resolveAgentFlag(allTargets []skills.AgentTarget, flag string) ([]skills.AgentTarget, error) {
	flag = strings.ToLower(strings.TrimSpace(flag))
	if flag == "all" {
		return allTargets, nil
	}
	for _, t := range allTargets {
		if t.Key == flag {
			return []skills.AgentTarget{t}, nil
		}
	}
	return nil, fmt.Errorf("unknown agent %q (valid: claude, codex, openclaw, all)", flag)
}

func runAgentPicker(allTargets []skills.AgentTarget, homeDir, action string) ([]skills.AgentTarget, error) {
	detected := skills.DetectAgents(allTargets)
	detectedKeys := make(map[string]bool)
	for _, d := range detected {
		detectedKeys[d.Key] = true
	}

	options := make([]huh.Option[string], len(allTargets))
	for i, t := range allTargets {
		label := fmt.Sprintf("%-12s (%s)", t.Name, skills.DisplayPath(t.BaseDir, homeDir))
		if action == "uninstall" && skills.IsInstalled(t) {
			v := skills.InstalledVersion(t)
			if v != "" {
				label += fmt.Sprintf(" — installed, %s", v)
			} else {
				label += " — installed"
			}
		}
		options[i] = huh.NewOption(label, t.Key)
	}

	var preselected []string
	if action == "uninstall" {
		for _, t := range allTargets {
			if skills.IsInstalled(t) {
				preselected = append(preselected, t.Key)
			}
		}
	} else {
		for _, t := range allTargets {
			if detectedKeys[t.Key] {
				preselected = append(preselected, t.Key)
			}
		}
		if len(preselected) == 0 {
			for _, t := range allTargets {
				preselected = append(preselected, t.Key)
			}
		}
	}

	selected := preselected
	title := fmt.Sprintf("%s healthsync skill for which AI agents?",
		strings.ToUpper(action[:1])+action[1:])

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(options...).
				Value(&selected).
				Validate(func(s []string) error {
					if len(s) == 0 {
						return fmt.Errorf("select at least one agent")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			fmt.Println("Cancelled.")
			return nil, nil
		}
		return nil, fmt.Errorf("selection error: %w", err)
	}

	selectedMap := make(map[string]bool)
	for _, s := range selected {
		selectedMap[s] = true
	}

	var targets []skills.AgentTarget
	for _, t := range allTargets {
		if selectedMap[t.Key] {
			targets = append(targets, t)
		}
	}

	return targets, nil
}

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
