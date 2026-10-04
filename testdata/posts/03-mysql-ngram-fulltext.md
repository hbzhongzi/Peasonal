---
title: MySQL 中文全文检索：ngram 的两个坑
slug: mysql-ngram-fulltext
tags: [MySQL, 全文检索]
published_at: 2026-09-15
---

# MySQL 中文全文检索：ngram 的两个坑

MySQL 8.0 内置了 ngram 分词器，中文全文检索可以开箱即用。但有两个坑，不提前知道就会得到「为什么搜不出来」的结论。

## 先建索引

```sql
ALTER TABLE posts
  ADD FULLTEXT INDEX ft_posts_search (title, content_md) WITH PARSER ngram;
```

查询：

```sql
SELECT id, title
FROM posts
WHERE MATCH (title, content_md) AGAINST ('并发模式' IN BOOLEAN MODE);
```

## 坑一：NATURAL LANGUAGE MODE 的 50% 阈值

不加 `IN BOOLEAN MODE` 时，MySQL 用的是自然语言模式，它会**忽略出现在超过 50% 行中的词**。

这个设计对「海量文档里的低频词」有意义，但对内容站是灾难：

- 文章总数少（比如 20 篇）时，稍微常见一点的词就会超过 50% 阈值
- 结果就是：搜「并发」有结果，搜「Go」没结果 —— 因为「Go」几乎每篇都出现

**所以必须显式写 `IN BOOLEAN MODE`。** 布尔模式不做阈值过滤，且支持 `+`、`-`、`*` 等操作符：

```sql
-- 必须含「并发」，不含「Python」
AGAINST ('+并发 -Python' IN BOOLEAN MODE)
```

## 坑二：ngram_token_size 是只读启动参数

ngram 按固定长度切词，`ngram_token_size` 默认 2，也就是「并发模式」会被切成 `并发`、`发模`、`模式` 三个 token。

想改成 1（支持单字搜索）时你会发现：

```sql
SET GLOBAL ngram_token_size = 1;
-- ERROR 1238 (HY000): Variable 'ngram_token_size' is a read only variable
```

它**只能在启动时设置**：

```ini
[mysqld]
ngram_token_size = 1
```

改完重启还不够 —— **已有的 FULLTEXT 索引是用旧的分词长度建的，必须重建**：

```sql
DROP INDEX ft_posts_search ON posts;
ALTER TABLE posts
  ADD FULLTEXT INDEX ft_posts_search (title, content_md) WITH PARSER ngram;
```

## 英文短词怎么办

ngram 是按字符切的，对英文短词效果很差。搜 "Go"（2 字符）在 `token_size=2` 下勉强可用，搜 "C"（1 字符）直接失效。

实用做法是**保留 LIKE 兜底**：先走 FULLTEXT，结果为空时退回子串匹配。

```sql
SELECT id, title FROM posts
WHERE title LIKE CONCAT('%', ?, '%') OR content_md LIKE CONCAT('%', ?, '%')
LIMIT 20;
```

在几百篇文章的量级下，全表 `LIKE` 是毫秒级的 —— 先跑起来，等真的慢了再换方案。

## 什么时候该换专门的搜索引擎

| 信号 | 说明 |
| --- | --- |
| 需要错别字容忍 | 「并法」要能搜到「并发」 |
| 需要拼音搜索 | 「bingfa」要能搜到「并发」 |
| 文章过万 | FULLTEXT 的排序质量开始不够用 |

出现上面任意一条，再引 Meilisearch 或 Typesense。在那之前，多运维一个服务是不划算的。
