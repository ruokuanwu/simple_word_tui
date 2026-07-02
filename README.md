# simpleword

一个基于 [bubbletea](https://github.com/charmbracelet/bubbletea) 的终端背单词程序，支持导入 Anki `.apkg` 单词本。

## 功能

- 导入 Anki `.apkg` 文件，自动解析单词、音标、释义与语音
- 数据存入 SQLite（`~/.simpleword/simpleword.db`），单词通过单词本 id 关联
- 背单词时展示单词 / 音标 / 自动播放语音，按掌握程度循环出题
- 每轮抽取若干单词，全部掌握后弹出祝贺框，可继续下一轮

## 目录结构（MVP 分层）

```
cmd/simpleword/       程序入口
internal/model/       数据模型（Model）
internal/store/       SQLite 持久化
internal/anki/        .apkg 解析库
internal/audio/       语音播放
internal/ui/          bubbletea 界面（View + Presenter）
```

## 运行

```bash
go run ./cmd/simpleword
```

## 操作

主界面：
- `w/s` 选择单词本
- `d` 进入背单词
- `f` 进入设置（学习情况 / 删除）或在 `+` 上触发导入
- `q` 退出

背单词：
- `d` 掌握，进入下一个
- `s` 查看完整释义
- `a` 上一个
- `w/x` 滚动释义
- `f` 跳过（不标记掌握）
- `q` 返回

导入：
- `ctrl+d` 确认导入
- `ctrl+q` 取消

## 语音

需系统中存在以下任一播放器：`ffplay`、`mpg123`、`paplay`、`aplay`、`afplay`。
