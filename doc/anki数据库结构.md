# Anki 数据库结构详解

## 概述

Anki 的 `.apkg` 文件本质上是一个 **ZIP 压缩包**，解压后包含以下主要文件：
- `collection.anki2` - SQLite 数据库文件（核心数据）
- `media` - JSON 文件，记录媒体文件映射
- 可能包含音频、图片等媒体文件（以数字命名，如 `0`, `1`, `2`）

`collection.anki2` 是一个 SQLite 数据库，包含了所有卡片、笔记、复习记录等数据。

---

## 数据库表结构

### 1. notes 表（笔记表）- 最核心的表

**用途**：存储所有卡片的内容数据（单词、释义、例句等）

```sql
CREATE TABLE notes (
    id              integer primary key,   /* 0 笔记唯一 ID */
    guid            text not null,         /* 1 全局唯一标识符 */
    mid             integer not null,      /* 2 模型 ID（卡片模板类型）*/
    mod             integer not null,      /* 3 最后修改时间（Unix 时间戳，秒）*/
    usn             integer not null,      /* 4 更新序列号（用于同步）*/
    tags            text not null,         /* 5 标签列表（空格分隔）*/
    flds            text not null,         /* 6 字段内容（\x1f 分隔）⭐ 最重要 */
    sfld            integer not null,      /* 7 排序字段（用于排序的首字段）*/
    csum            integer not null,      /* 8 校验和（用于查重）*/
    flags           integer not null,      /* 9 标志位 */
    data            text not null          /* 10 附加数据 */
);
```

**重点字段说明**：

- **`flds`**（fields）：这是最核心的字段！
  - 存储卡片的所有内容字段
  - 使用 `\x1f`（ASCII 31，Unit Separator）作为分隔符
  - 例如：`"apple\x1f苹果\x1fI like apple."`
  - 字段顺序由模型定义，通常第一个是"正面"（如单词），第二个是"背面"（如释义）

- **`mid`**：模型 ID，关联到 `col` 表的 `models` 字段
  - 定义了这个笔记使用什么模板（如"基础"、"基础带反向"等）

- **`tags`**：标签，用空格分隔，如 `"词汇 四级 重点"`

**索引**：
```sql
CREATE INDEX ix_notes_usn on notes (usn);   -- 用于同步
CREATE INDEX ix_notes_csum on notes (csum); -- 用于查重
```

---

### 2. cards 表（卡片表）

**用途**：存储每张卡片的学习状态和调度信息

一个 note 可以生成多张 card（例如正向卡片和反向卡片）

```sql
CREATE TABLE cards (
    id              integer primary key,   /* 0 卡片唯一 ID */
    nid             integer not null,      /* 1 对应的笔记 ID ⭐ 关联 notes.id */
    did             integer not null,      /* 2 牌组 ID（deck ID）*/
    ord             integer not null,      /* 3 卡片序号（同一笔记的第几张卡）*/
    mod             integer not null,      /* 4 最后修改时间 */
    usn             integer not null,      /* 5 更新序列号 */
    type            integer not null,      /* 6 卡片类型（0=新卡,1=学习中,2=复习）*/
    queue           integer not null,      /* 7 队列类型（-1=暂停,0=新卡,1=学习,2=复习）*/
    due             integer not null,      /* 8 到期时间/位置 */
    ivl             integer not null,      /* 9 间隔天数（interval）*/
    factor          integer not null,      /* 10 难度系数（ease factor，千分比）*/
    reps            integer not null,      /* 11 复习次数 */
    lapses          integer not null,      /* 12 遗忘次数（失败次数）*/
    left            integer not null,      /* 13 当日剩余学习次数 */
    odue            integer not null,      /* 14 原到期时间（筛选/临时牌组用）*/
    odid            integer not null,      /* 15 原牌组 ID */
    flags           integer not null,      /* 16 标志位（用于标记卡片）*/
    data            text not null          /* 17 附加数据 */
);
```

**重点字段说明**：

- **`nid`**：关联到 `notes` 表，通过这个字段可以获取卡片内容
- **`ord`**：卡片序号
  - `0` = 第一张卡片（通常是正向：单词 → 释义）
  - `1` = 第二张卡片（通常是反向：释义 → 单词）
- **`type` 和 `queue`**：表示卡片当前的学习状态
- **`due`**：下次复习时间，对于新卡片是位置序号
- **`ivl`**：间隔天数，Anki 的核心调度参数
- **`factor`**：难度系数，默认 2500（即 2.5 倍），表示下次间隔的增长倍数

**索引**：
```sql
CREATE INDEX ix_cards_usn on cards (usn);
CREATE INDEX ix_cards_nid on cards (nid);          -- 用于关联笔记
CREATE INDEX ix_cards_sched on cards (did, queue, due); -- 用于调度查询
```

---

### 3. col 表（集合配置表）

**用途**：存储整个集合（collection）的全局配置和元数据

这是一个**单行表**，只有一条记录。

```sql
CREATE TABLE col (
    id              integer primary key,   -- 集合 ID（通常是创建时间戳）
    crt             integer not null,      -- 创建时间（Unix 时间戳，秒）
    mod             integer not null,      -- 最后修改时间（Unix 时间戳，毫秒）
    scm             integer not null,      -- Schema 修改时间（毫秒）
    ver             integer not null,      -- 版本号
    dty             integer not null,      -- Dirty（需要同步标志）
    usn             integer not null,      -- 更新序列号
    ls              integer not null,      -- 最后同步时间
    conf            text not null,         -- 全局配置（JSON）
    models          text not null,         -- 笔记模型定义（JSON）⭐
    decks           text not null,         -- 牌组定义（JSON）⭐
    dconf           text not null,         -- 牌组配置（JSON）
    tags            text not null          -- 标签列表（JSON）
);
```

**重点字段说明**：

- **`models`**：JSON 格式，定义所有笔记类型的模板
  ```json
  {
    "1234567890": {
      "id": 1234567890,
      "name": "基础",
      "flds": [
        {"name": "Front", "ord": 0},
        {"name": "Back", "ord": 1}
      ],
      ...
    }
  }
  ```

- **`decks`**：JSON 格式，定义所有牌组信息
  ```json
  {
    "1": {
      "id": 1,
      "name": "默认",
      ...
    }
  }
  ```

---

### 4. revlog 表（复习日志表）

**用途**：记录每次复习的详细历史

```sql
CREATE TABLE revlog (
    id              integer primary key,   -- 日志 ID（时间戳+随机数）
    cid             integer not null,      -- 卡片 ID
    usn             integer not null,      -- 更新序列号
    ease            integer not null,      -- 按钮选择（1=再来,2=困难,3=良好,4=简单）
    ivl             integer not null,      -- 复习后的间隔（天数）
    lastIvl         integer not null,      -- 复习前的间隔
    factor          integer not null,      -- 新的难度系数
    time            integer not null,      -- 耗时（毫秒）
    type            integer not null       -- 复习类型（0=学习,1=复习,2=重学,3=筛选）
);
```

**用途**：用于统计学习效率、分析遗忘曲线等

---

### 5. graves 表（墓碑表）

**用途**：记录已删除的对象，用于同步时告知其他设备删除

```sql
CREATE TABLE graves (
    usn             integer not null,      -- 更新序列号
    oid             integer not null,      -- 原对象 ID
    type            integer not null       -- 对象类型（0=卡片,1=笔记,2=牌组）
);
```

---

## 表之间的关系

```
┌─────────┐
│   col   │  (全局配置)
│  单行表  │  - models: 定义笔记类型
└─────────┘  - decks: 定义牌组

     ↓ 引用 mid (模型 ID)

┌─────────┐
│  notes  │  (笔记/内容)
│ id, flds│  - flds: 单词和释义的实际内容
└─────────┘

     ↓ 一对多关系 (nid)

┌─────────┐
│  cards  │  (卡片/学习状态)
│nid, due │  - 每个 note 可生成多张 card
└─────────┘  - 记录学习进度和调度信息

     ↓ 一对多关系 (cid)

┌─────────┐
│ revlog  │  (复习历史)
│ cid, ease│  - 每次复习产生一条记录
└─────────┘
```

---

## 对于背单词应用的关键查询

### 1. 获取所有单词和释义

```sql
SELECT 
    id,
    flds  -- 需要解析 \x1f 分隔的字段
FROM notes
LIMIT 1000;
```

处理示例（Dart）：
```dart
final fields = flds.split('\x1f');
final word = fields[0];           // 第一个字段：单词
final definition = fields[1];     // 第二个字段：释义
```

### 2. 获取卡片的学习状态

```sql
SELECT 
    c.id,
    c.nid,
    c.type,
    c.queue,
    c.reps,
    n.flds
FROM cards c
JOIN notes n ON c.nid = n.id
WHERE c.queue >= 0  -- 排除已暂停的卡片
ORDER BY c.due;
```

### 3. 统计学习进度

```sql
-- 新卡片数量
SELECT COUNT(*) FROM cards WHERE type = 0;

-- 待复习卡片数量
SELECT COUNT(*) FROM cards WHERE type = 2 AND queue = 2;

-- 总复习次数
SELECT SUM(reps) FROM cards;
```

---

## 字段值说明

### type 和 queue 的含义

**type**（卡片类型）：
- `0` = 新卡片（new）
- `1` = 学习中（learning）
- `2` = 复习卡片（review）
- `3` = 重新学习（relearning）

**queue**（队列类型）：
- `-3` = 用户手动埋藏
- `-2` = 调度器埋藏
- `-1` = 暂停
- `0` = 新卡片队列
- `1` = 学习队列
- `2` = 复习队列
- `3` = 在学习中，但由于日期变更需要重新处理
- `4` = 预习队列

---

## MVP 应用的简化方案

对于我们的 MVP 版本，可以只关注 `notes` 表：

1. 读取 `notes.flds` 字段
2. 按 `\x1f` 分隔获取单词和释义
3. 在应用内存中管理学习状态（不需要读写 cards 和 revlog）

这样可以：
- 简化实现复杂度
- 避免修改原始 apkg 文件
- 快速实现 MVP 功能

未来扩展时可以：
- 读取 `cards` 表获取历史学习记录
- 根据 `type`、`queue`、`due` 实现更智能的调度
- 根据 `revlog` 分析学习效率