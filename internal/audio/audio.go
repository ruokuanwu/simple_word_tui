// Package audio 提供简单的异步音频播放能力。
package audio

import (
	"os/exec"
)

// player 缓存检测到的可用播放命令。
var player = detectPlayer()

// candidates 是按优先级排列的播放器命令。
var candidates = []struct {
	cmd  string
	args func(file string) []string
}{
	{"ffplay", func(f string) []string { return []string{"-nodisp", "-autoexit", "-loglevel", "quiet", f} }},
	{"mpg123", func(f string) []string { return []string{"-q", f} }},
	{"paplay", func(f string) []string { return []string{f} }},
	{"aplay", func(f string) []string { return []string{"-q", f} }},
	{"afplay", func(f string) []string { return []string{f} }},
}

type playerCmd struct {
	bin  string
	args func(string) []string
}

// detectPlayer 在系统中查找第一个可用的播放命令。
func detectPlayer() *playerCmd {
	for _, c := range candidates {
		if path, err := exec.LookPath(c.cmd); err == nil {
			return &playerCmd{bin: path, args: c.args}
		}
	}
	return nil
}

// Available 报告系统是否存在可用的播放器。
func Available() bool {
	return player != nil
}

// Play 异步播放指定音频文件。文件为空或无可用播放器时静默忽略。
func Play(file string) {
	if file == "" || player == nil {
		return
	}
	cmd := exec.Command(player.bin, player.args(file)...)
	// 异步执行，不阻塞 UI；播放结束由进程自行退出。
	_ = cmd.Start()
	go func() { _ = cmd.Wait() }()
}
