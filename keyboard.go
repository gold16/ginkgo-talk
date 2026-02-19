package main

import (
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	procSendInput = user32.NewProc("SendInput")
)

const (
	inputKBD         = 1
	keyeventfUnicode = 0x0004
	keyeventfKeyup   = 0x0002

	// Size of INPUT struct on 64-bit Windows = 40 bytes
	// type(4) + padding(4) + union(32)
	inputSize64 = 40
	// Size of INPUT struct on 32-bit Windows = 28 bytes
	// type(4) + union(24)
	inputSize32 = 28
)

// inputSize returns the correct size of the INPUT struct for the current architecture.
func inputSize() uintptr {
	if unsafe.Sizeof(uintptr(0)) == 8 {
		return inputSize64
	}
	return inputSize32
}

// makeKeyInput creates a raw byte slice representing a KEYBOARD INPUT struct.
// This avoids Go struct alignment issues by manually laying out the bytes.
func makeKeyInput(wVk uint16, wScan uint16, dwFlags uint32) []byte {
	size := inputSize()
	buf := make([]byte, size)

	// Type = INPUT_KEYBOARD (1) at offset 0
	buf[0] = byte(inputKBD)
	buf[1] = 0
	buf[2] = 0
	buf[3] = 0

	// Union starts at offset 4 (32-bit) or offset 8 (64-bit due to alignment)
	var unionOffset uintptr
	if size == inputSize64 {
		unionOffset = 8
	} else {
		unionOffset = 4
	}

	// KEYBDINPUT layout within the union:
	// wVk:         offset 0, size 2
	// wScan:       offset 2, size 2
	// dwFlags:     offset 4, size 4
	// time:        offset 8, size 4
	// dwExtraInfo: offset 16 (64-bit) or offset 12 (32-bit), size pointer
	o := unionOffset
	buf[o+0] = byte(wVk)
	buf[o+1] = byte(wVk >> 8)
	buf[o+2] = byte(wScan)
	buf[o+3] = byte(wScan >> 8)
	buf[o+4] = byte(dwFlags)
	buf[o+5] = byte(dwFlags >> 8)
	buf[o+6] = byte(dwFlags >> 16)
	buf[o+7] = byte(dwFlags >> 24)
	// time = 0 (already zeroed)
	// dwExtraInfo = 0 (already zeroed)

	return buf
}

// TypeText simulates keyboard input for the given Unicode string.
// It uses SendInput with KEYEVENTF_UNICODE to support any character including CJK.
func TypeText(text string) error {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	size := inputSize()
	log.Printf("⌨️  SendInput struct size: %d bytes, typing %d characters", size, len(runes))

	// Process in chunks to avoid overwhelming the input queue
	chunkSize := 20 // characters per chunk
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := runes[start:end]

		if err := typeChunk(chunk, size); err != nil {
			return err
		}

		// Small delay between chunks
		if end < len(runes) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}

func typeChunk(runes []rune, size uintptr) error {
	// Build the raw input buffer: 2 events per rune (key down + key up)
	var inputs []byte
	var count int

	for _, r := range runes {
		// Handle newline: use Shift+Enter (new line without submit)
		if r == '\n' {
			inputs = append(inputs,
				makeKeyInput(0x10, 0, 0)..., // Shift down
			)
			inputs = append(inputs,
				makeKeyInput(0x0D, 0, 0)..., // Enter down
			)
			inputs = append(inputs,
				makeKeyInput(0x0D, 0, keyeventfKeyup)..., // Enter up
			)
			inputs = append(inputs,
				makeKeyInput(0x10, 0, keyeventfKeyup)..., // Shift up
			)
			count += 4
			continue
		}
		// Unicode character
		inputs = append(inputs, makeKeyInput(0, uint16(r), keyeventfUnicode)...)
		inputs = append(inputs, makeKeyInput(0, uint16(r), keyeventfUnicode|keyeventfKeyup)...)
		count += 2
	}

	if count == 0 {
		return nil
	}

	ret, _, err := procSendInput.Call(
		uintptr(count),
		uintptr(unsafe.Pointer(&inputs[0])),
		size,
	)

	if ret == 0 {
		return fmt.Errorf("SendInput failed (sent 0/%d events): %w", count, err)
	}
	if int(ret) != count {
		log.Printf("⚠️  SendInput: only %d/%d events accepted", ret, count)
	}

	return nil
}

// SelectAllAndDelete sends Ctrl+A then Delete to clear the focused input field.
func SelectAllAndDelete() error {
	size := inputSize()

	// VK codes
	const (
		vkControl = 0x11
		vkA       = 0x41
		vkDelete  = 0x2E
	)

	var inputs []byte

	// Ctrl down
	inputs = append(inputs, makeKeyInput(vkControl, 0, 0)...)
	// A down
	inputs = append(inputs, makeKeyInput(vkA, 0, 0)...)
	// A up
	inputs = append(inputs, makeKeyInput(vkA, 0, keyeventfKeyup)...)
	// Ctrl up
	inputs = append(inputs, makeKeyInput(vkControl, 0, keyeventfKeyup)...)
	// Delete down
	inputs = append(inputs, makeKeyInput(vkDelete, 0, 0)...)
	// Delete up
	inputs = append(inputs, makeKeyInput(vkDelete, 0, keyeventfKeyup)...)

	count := 6
	ret, _, err := procSendInput.Call(
		uintptr(count),
		uintptr(unsafe.Pointer(&inputs[0])),
		size,
	)

	if ret == 0 {
		return fmt.Errorf("SendInput (clear) failed: %w", err)
	}
	return nil
}

// PressEnter sends an Enter key press.
func PressEnter() error {
	return pressKey(0x0D, false) // VK_RETURN
}

// PressShiftEnter sends Shift+Enter key press (new line in many editors).
func PressShiftEnter() error {
	size := inputSize()
	const (
		vkShift  = 0x10
		vkReturn = 0x0D
	)
	var inputs []byte
	inputs = append(inputs, makeKeyInput(vkShift, 0, 0)...)
	inputs = append(inputs, makeKeyInput(vkReturn, 0, 0)...)
	inputs = append(inputs, makeKeyInput(vkReturn, 0, keyeventfKeyup)...)
	inputs = append(inputs, makeKeyInput(vkShift, 0, keyeventfKeyup)...)

	ret, _, err := procSendInput.Call(
		uintptr(4),
		uintptr(unsafe.Pointer(&inputs[0])),
		size,
	)
	if ret == 0 {
		return fmt.Errorf("SendInput (shift+enter) failed: %w", err)
	}
	return nil
}

// pressKey sends a single key press (down + up).
func pressKey(vk uint16, extended bool) error {
	size := inputSize()
	var inputs []byte
	inputs = append(inputs, makeKeyInput(vk, 0, 0)...)
	inputs = append(inputs, makeKeyInput(vk, 0, keyeventfKeyup)...)

	ret, _, err := procSendInput.Call(
		uintptr(2),
		uintptr(unsafe.Pointer(&inputs[0])),
		size,
	)
	if ret == 0 {
		return fmt.Errorf("SendInput (key 0x%X) failed: %w", vk, err)
	}
	return nil
}

// pressCtrlKey sends Ctrl+<key> combo.
func pressCtrlKey(vk uint16) error {
	size := inputSize()
	const vkControl = 0x11
	var inputs []byte
	inputs = append(inputs, makeKeyInput(vkControl, 0, 0)...)
	inputs = append(inputs, makeKeyInput(vk, 0, 0)...)
	inputs = append(inputs, makeKeyInput(vk, 0, keyeventfKeyup)...)
	inputs = append(inputs, makeKeyInput(vkControl, 0, keyeventfKeyup)...)

	ret, _, err := procSendInput.Call(
		uintptr(4),
		uintptr(unsafe.Pointer(&inputs[0])),
		size,
	)
	if ret == 0 {
		return fmt.Errorf("SendInput (ctrl+0x%X) failed: %w", vk, err)
	}
	return nil
}

// PressCtrlZ sends Ctrl+Z (undo).
func PressCtrlZ() error {
	return pressCtrlKey(0x5A) // VK_Z
}

// PressCtrlV sends Ctrl+V (paste).
func PressCtrlV() error {
	return pressCtrlKey(0x56) // VK_V
}

// PressTab sends a Tab key press.
func PressTab() error {
	return pressKey(0x09, false) // VK_TAB
}

// PressEscape sends an Escape key press.
func PressEscape() error {
	return pressKey(0x1B, false) // VK_ESCAPE
}
