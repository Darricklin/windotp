//go:build darwin

package typer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

const frontContextScript = `const systemEvents = Application("System Events");
const windTerm = systemEvents.applicationProcesses.byName("WindTerm");
if (!windTerm.exists()) {
    throw new Error("WindTerm is not running");
}
if (!windTerm.frontmost()) {
    throw new Error("WindTerm is not the frontmost application");
}

function safe(call, fallback) {
    try {
        return call();
    } catch (_) {
        return fallback;
    }
}

const windows = safe(() => windTerm.windows(), []);
if (windows.length === 0) {
    throw new Error("WindTerm has no accessible window");
}
const frontWindow = windows[0];
const candidates = [];
const seen = {};
let visited = 0;
let matched = false;

function matchesProfile(value) {
    if (typeof value !== "string") {
        return false;
    }
    const normalized = value.toLowerCase();
    for (let i = 0; i < profileMatches.length; i += 1) {
        if (normalized.indexOf(profileMatches[i].toLowerCase()) !== -1) {
            return true;
        }
    }
    return false;
}

function addCandidate(value) {
    if (typeof value !== "string" || value.length === 0 || seen[value]) {
        return;
    }
    seen[value] = true;
    candidates.push(value);
    if (matchesProfile(value)) {
        matched = true;
    }
}

function elementTexts(element, includeValue) {
    const values = [
        safe(() => element.name(), ""),
        safe(() => element.title(), ""),
        safe(() => element.description(), ""),
        safe(() => element.help(), "")
    ];
    if (includeValue) {
        values.push(safe(() => element.value(), ""));
    }
    const texts = [];
    for (let i = 0; i < values.length; i += 1) {
        if (typeof values[i] === "string" && values[i].length > 0) {
            texts.push(values[i]);
        }
    }
    return texts;
}

function addElementCandidates(element, includeValue) {
    const texts = elementTexts(element, includeValue);
    for (let i = 0; i < texts.length; i += 1) {
        addCandidate(texts[i]);
    }
    return texts;
}

const windowTexts = [
    safe(() => frontWindow.name(), ""),
    safe(() => frontWindow.title(), "")
].filter((value) => typeof value === "string" && value.length > 0);
const windowName = windowTexts.length > 0 ? windowTexts[0] : "";
for (let i = 0; i < windowTexts.length; i += 1) {
    addCandidate(windowTexts[i]);
}

const queue = [{element: frontWindow, depth: 0, labelDepth: 0}];
const menuBars = safe(() => windTerm.menuBars(), []);
for (let i = 0; i < menuBars.length; i += 1) {
    queue.push({element: menuBars[i], depth: 0, labelDepth: 0});
}
let cursor = 0;
while (!matched && cursor < queue.length && visited < 1200) {
    const current = queue[cursor];
    cursor += 1;
    const element = current.element;
    const depth = current.depth;
    const labelDepth = current.labelDepth;
    visited += 1;
    const role = safe(() => element.role(), "");
    const focused = safe(() => element.focused(), false) === true;
    const selected = safe(() => element.selected(), false) === true;
    let activeControl = false;
    if (role === "AXRadioButton" || role === "AXTab" || role === "AXButton" || role === "AXCheckBox" || role === "AXMenuItem") {
        const value = safe(() => element.value(), 0);
        activeControl = value === 1 || value === true || value === "1";
    }
    if (role === "AXMenuItem" && !activeControl) {
        const mark = safe(() => element.attributes.byName("AXMenuItemMarkChar").value(), "");
        activeControl = typeof mark === "string" && mark.length > 0;
    }
    if (focused || selected || activeControl || labelDepth > 0) {
        const includeValue = role !== "AXTextArea" && role !== "AXWebArea";
        addElementCandidates(element, includeValue);
    }
    const skipChildren = role === "AXTextArea" || role === "AXTable" || role === "AXOutline" || role === "AXWebArea";
    if (!matched && !skipChildren && depth < 12) {
        const children = safe(() => element.uiElements(), []);
        let childLabelDepth = labelDepth > 0 ? labelDepth - 1 : 0;
        if (selected || activeControl) {
            childLabelDepth = 3;
        }
        for (let i = 0; i < children.length; i += 1) {
            queue.push({element: children[i], depth: depth + 1, labelDepth: childLabelDepth});
        }
    }
}

JSON.stringify({
    window: windowName,
    candidates: candidates
});`

const promptVisibleScript = `const systemEvents = Application("System Events");
const windTerm = systemEvents.applicationProcesses.byName("WindTerm");

if (!windTerm.exists() || !windTerm.frontmost()) {
    JSON.stringify({prompt: false, input: false, rememberFound: false, rememberCleared: false});
} else {
    function safe(call, fallback) {
        try {
            return call();
        } catch (_) {
            return fallback;
        }
    }

    function isMFAInput(element) {
        const role = safe(() => element.role(), "");
        if (role === "AXTextArea" || role === "AXWebArea") {
            return false;
        }
        const editable = safe(() => element.attributes.byName("AXEditable").value(), false) === true;
        return editable || role === "AXTextField" || role === "AXSecureTextField" || role === "AXComboBox";
    }

    function isChecked(value) {
        return value === true || value === 1 || value === "1" || value === "true" || value === "checked";
    }

    function elementValues(element) {
        return [
            safe(() => element.name(), ""),
            safe(() => element.description(), ""),
            safe(() => element.value(), "")
        ];
    }

    function isRememberCheckbox(element, role) {
        if (role !== "AXCheckBox") {
            return false;
        }
        const values = [
            safe(() => element.name(), ""),
            safe(() => element.title(), ""),
            safe(() => element.description(), ""),
            safe(() => element.help(), "")
        ];
        for (let i = 0; i < values.length; i += 1) {
            if (typeof values[i] !== "string") {
                continue;
            }
            const normalized = values[i].toLowerCase();
            if (normalized.indexOf("remember this step") !== -1 || values[i].indexOf("记住这一步") !== -1) {
                return true;
            }
        }
        return false;
    }

    function clickCenter(element) {
        const position = safe(() => element.position(), []);
        const size = safe(() => element.size(), []);
        if (position.length !== 2 || size.length !== 2) {
            return false;
        }
        return safe(() => {
            systemEvents.click({at: [
                Math.round(position[0] + size[0] / 2),
                Math.round(position[1] + size[1] / 2)
            ]});
            return true;
        }, false);
    }

    function checked(element) {
        return isChecked(safe(() => element.value(), true));
    }

    function waitForCheckboxUpdate() {
        delay(0.05);
    }

    let promptFound = false;
    let rememberFound = false;
    let rememberCleared = false;
    const focusedInput = safe(() => windTerm.attributes.byName("AXFocusedUIElement").value(), null);
    // The terminal buffer is normally the focused AXTextArea. Ignoring it
    // makes checks before the MFA dialog appears effectively constant-time.
    const inputFound = focusedInput !== null && isMFAInput(focusedInput);

    function focusMFAInput() {
        if (safe(() => focusedInput.focused(), false) === true) {
            return true;
        }
        safe(() => {
            focusedInput.attributes.byName("AXFocused").value = true;
            return true;
        }, false);
        if (safe(() => focusedInput.focused(), false) === true) {
            return true;
        }
        clickCenter(focusedInput);
        return safe(() => focusedInput.focused(), false) === true;
    }

    function clearRememberCheckbox(element) {
        if (!checked(element)) {
            return true;
        }

        safe(() => element.actions.byName("AXPress").perform(), null);
        waitForCheckboxUpdate();
        if (!checked(element)) {
            return focusMFAInput();
        }

        safe(() => element.click(), null);
        waitForCheckboxUpdate();
        if (!checked(element)) {
            return focusMFAInput();
        }

        safe(() => {
            element.attributes.byName("AXFocused").value = true;
            systemEvents.keyCode(49);
        }, null);
        waitForCheckboxUpdate();
        if (!checked(element)) {
            return focusMFAInput();
        }

        clickCenter(element);
        waitForCheckboxUpdate();
        if (!checked(element)) {
            return focusMFAInput();
        }
        focusMFAInput();
        return false;
    }

    function inspect(element, role) {
        if (isRememberCheckbox(element, role)) {
            rememberFound = true;
            rememberCleared = clearRememberCheckbox(element);
        }

        // Ignore text rendered in the terminal buffer. This command is only
        // for WindTerm's graphical keyboard-interactive dialog.
        if (!promptFound && role !== "AXTextArea" && role !== "AXWebArea") {
            const values = elementValues(element);
            for (let i = 0; i < values.length; i += 1) {
                if (typeof values[i] === "string" && values[i].indexOf(expectedPrompt) !== -1) {
                    promptFound = true;
                    break;
                }
            }
        }
    }

    function scan(root, maxDepth, maxVisited) {
        const queue = [{element: root, depth: 0}];
        let cursor = 0;
        let visited = 0;
        while (cursor < queue.length && visited < maxVisited && !(promptFound && rememberCleared)) {
            const current = queue[cursor];
            cursor += 1;
            visited += 1;

            const element = current.element;
            const depth = current.depth;
            const role = safe(() => element.role(), "");
            inspect(element, role);

            const skipChildren = role === "AXTextArea" || role === "AXTable" || role === "AXOutline" || role === "AXWebArea";
            if (!skipChildren && depth < maxDepth) {
                const children = safe(() => element.uiElements(), []);
                for (let i = 0; i < children.length; i += 1) {
                    queue.push({element: children[i], depth: depth + 1});
                }
            }
        }
    }

    if (inputFound) {
        const roots = [];
        let current = safe(() => focusedInput.attributes.byName("AXParent").value(), null);
        for (let depth = 0; current !== null && depth < 10; depth += 1) {
            roots.push(current);
            const role = safe(() => current.role(), "");
            if (role === "AXSheet" || role === "AXDialog" || role === "AXWindow") {
                break;
            }
            current = safe(() => current.attributes.byName("AXParent").value(), null);
        }

        // Search the nearest input ancestors first. The prompt and checkbox
        // are normally siblings within the same small dialog container.
        for (let i = 0; i < roots.length && !(promptFound && rememberCleared); i += 1) {
            scan(roots[i], 4, 160);
        }
    }

    JSON.stringify({
        prompt: promptFound,
        input: inputFound,
        rememberFound: rememberFound,
        rememberCleared: rememberCleared
    });
}`

func platformType(code string, opts Options) error {
	if !sixDigits.MatchString(code) {
		return fmt.Errorf("refusing to type a value that is not exactly six digits")
	}
	guard := `if name of frontProcess is not "WindTerm" then error "WindTerm is not the frontmost application"`
	if opts.AllowAnyApp {
		guard = ""
	}
	activate := ""
	if opts.ActivateWindTerm {
		activate = `set windTermProcess to first application process whose name is "WindTerm"
set frontmost of windTermProcess to true
delay 0.2`
	}
	enter := ""
	if opts.Enter {
		enter = "key code 36"
	}
	script := fmt.Sprintf(`tell application "System Events"
%s
set frontProcess to first application process whose frontmost is true
%s
keystroke "%s"
%s
end tell
`, activate, guard, code, enter)
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("type into WindTerm: %s: %w", bytes.TrimSpace(output), err)
	}
	return nil
}

func platformChoose(profiles []string, defaultProfile string) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("no profiles configured")
	}
	quoted := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		quoted = append(quoted, `"`+profile+`"`)
	}
	defaultClause := ""
	if defaultProfile != "" {
		defaultClause = ` default items {"` + defaultProfile + `"}`
	}
	script := fmt.Sprintf(`tell application "System Events"
set frontProcess to first application process whose frontmost is true
if name of frontProcess is not "WindTerm" then error "WindTerm is not the frontmost application"
end tell
set picked to choose from list {%s} with title "WindOTP" with prompt "Select a JumpServer"%s
if picked is false then return ""
return item 1 of picked
`, strings.Join(quoted, ", "), defaultClause)
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("choose JumpServer profile: %s: %w", bytes.TrimSpace(output), err)
	}
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", ErrCanceled
	}
	return selected, nil
}

func platformContext(matches []string) (FrontContext, error) {
	encodedMatches, err := json.Marshal(matches)
	if err != nil {
		return FrontContext{}, fmt.Errorf("encode profile matches: %w", err)
	}
	cmd := exec.Command("/usr/bin/osascript", "-l", "JavaScript")
	cmd.Stdin = bytes.NewBufferString("const profileMatches = " + string(encodedMatches) + ";\n" + frontContextScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return FrontContext{}, fmt.Errorf("inspect active WindTerm tab: %s: %w", bytes.TrimSpace(output), err)
	}
	var context FrontContext
	if err := json.Unmarshal(bytes.TrimSpace(output), &context); err != nil {
		return FrontContext{}, fmt.Errorf("decode active WindTerm context: %w", err)
	}
	return context, nil
}

func platformPromptVisible(prompt string) (bool, error) {
	encodedPrompt, err := json.Marshal(prompt)
	if err != nil {
		return false, fmt.Errorf("encode MFA prompt: %w", err)
	}
	cmd := exec.Command("/usr/bin/osascript", "-l", "JavaScript")
	cmd.Stdin = bytes.NewBufferString("const expectedPrompt = " + string(encodedPrompt) + ";\n" + promptVisibleScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("inspect WindTerm MFA dialog: %s: %w", bytes.TrimSpace(output), err)
	}
	var result struct {
		Prompt          bool `json:"prompt"`
		Input           bool `json:"input"`
		RememberFound   bool `json:"rememberFound"`
		RememberCleared bool `json:"rememberCleared"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &result); err != nil {
		return false, fmt.Errorf("decode WindTerm MFA dialog state: %w", err)
	}
	if result.Prompt && result.Input && result.RememberFound && !result.RememberCleared {
		return false, fmt.Errorf("WindTerm MFA dialog is ready, but cannot clear Remember this step; grant Accessibility access and retry")
	}
	return result.Prompt && result.Input && result.RememberFound && result.RememberCleared, nil
}

func platformCheck() error {
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(`tell application "System Events" to get name of first application process whose frontmost is true`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("macOS Accessibility check failed: %s: %w", bytes.TrimSpace(output), err)
	}
	return nil
}
