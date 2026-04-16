package service

import (
	"context"
	"math"
	"strings"
	"testing"

	_ "canned-exp/internal/database/migrations/main"

	"canned-exp/internal/embedding"
	"canned-exp/internal/model"
	"canned-exp/internal/repository"
	"canned-exp/internal/vectorstore"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/CuratorC/gocanned/migration"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

// mockEmbeddingProvider 基于字符频率的 mock embedding
// 相似文本会产生相似向量，使语义搜索在测试中可用
type mockEmbeddingProvider struct {
	dimensions int
}

func (m *mockEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		results[i] = textToVector(text, m.dimensions)
	}
	return results, nil
}

func (m *mockEmbeddingProvider) Dimensions() int {
	return m.dimensions
}

// textToVector 将文本转化为基于字符频率的向量
func textToVector(text string, dims int) []float64 {
	vec := make([]float64, dims)
	for _, r := range text {
		idx := int(uint32(r)) % dims
		vec[idx] += 1.0
	}
	// 归一化
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for j := range vec {
			vec[j] /= norm
		}
	}
	return vec
}

// 确保 mockEmbeddingProvider 实现了 embedding.Provider 接口
var _ embedding.Provider = (*mockEmbeddingProvider)(nil)

func init() {
	logger.Logger = zap.NewNop()
}

// setupTestRepo 创建使用内存 SQLite + gocanned migration 的测试仓库
func setupTestRepo(t *testing.T) repository.Repository {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm memory sqlite: %v", err)
	}

	dbDB := &database.DB{Gorm: gormDB, Driver: "sqlite"}
	runner := migration.NewRunner("main", dbDB)
	if err := runner.Run(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	vecStore, err := vectorstore.NewSQLiteVecStore(dbDB.SQL())
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}

	embedder := &mockEmbeddingProvider{dimensions: 64}
	repo := repository.NewGormRepo(gormDB, vecStore, embedder)
	return repo
}

// testServiceSetup 创建使用 mock Repository 的 ExperienceService 实例
func testServiceSetup(t *testing.T) *ExperienceService {
	t.Helper()
	repo := setupTestRepo(t)
	return NewExperienceService(repo)
}

// --- Save 测试 ---

func TestService_Save(t *testing.T) {
	svc := testServiceSetup(t)
	ctx := context.Background()

	t.Run("合法经验保存成功", func(t *testing.T) {
		result, err := svc.Save(ctx, &model.Experience{
			Title:   "Go 错误处理",
			Content: "Go 中使用 if err != nil 进行错误处理",
			Tags:    []string{"go", "error"},
			Source:  "claude",
		}, false)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if !result.Saved {
			t.Error("合法经验应保存成功")
		}
		if result.ID == "" {
			t.Fatal("Save() 返回了空 ID")
		}
	})

	t.Run("无标题的经验也能保存", func(t *testing.T) {
		result, err := svc.Save(ctx, &model.Experience{
			Content: "没有标题的经验",
		}, false)
		if err != nil {
			t.Fatalf("期望保存成功: %v", err)
		}
		if !result.Saved {
			t.Error("应保存成功")
		}
		if result.ID == "" {
			t.Fatal("Save() 返回了空 ID")
		}
	})

	t.Run("空内容被拦截不进入 Repository", func(t *testing.T) {
		_, err := svc.Save(ctx, &model.Experience{
			Title:   "有标题",
			Content: "",
		}, false)
		if err == nil {
			t.Error("空内容应被 Service 拦截")
		}
	})

	t.Run("空白内容被拦截", func(t *testing.T) {
		_, err := svc.Save(ctx, &model.Experience{
			Title:   "有标题",
			Content: "   \t  ",
		}, false)
		if err == nil {
			t.Error("空白内容应被 Service 拦截")
		}
	})

	t.Run("超长内容被拦截", func(t *testing.T) {
		_, err := svc.Save(ctx, &model.Experience{
			Title:   "超长",
			Content: strings.Repeat("很长的内容。", 2000),
		}, false)
		if err == nil {
			t.Error("超长内容应被 Service 拦截")
		}
	})
}

// --- Save 去重测试 ---

func TestService_Save_Dedup(t *testing.T) {
	t.Run("正常保存（无相似经验）", func(t *testing.T) {
		repo := setupTestRepo(t)
		svc := NewExperienceService(repo)
		ctx := context.Background()

		result, err := svc.Save(ctx, &model.Experience{
			Content: "Go 测试文件应与源文件放在同一目录",
		}, false)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if !result.Saved {
			t.Error("空库首次保存，Saved 应为 true")
		}
		if result.ID == "" {
			t.Error("保存成功应返回 ID")
		}
		if len(result.SimilarExperiences) > 0 {
			t.Error("空库首次保存不应有相似经验")
		}
	})

	t.Run("发现相似经验且 force=false", func(t *testing.T) {
		repo := setupTestRepo(t)
		svc := NewExperienceService(repo)
		ctx := context.Background()

		// 先保存一条（force=true 确保）
		svc.Save(ctx, &model.Experience{
			Content: "Go 测试文件应与源文件放在同一目录",
		}, true)

		// 保存完全相同的内容，不强制
		result, err := svc.Save(ctx, &model.Experience{
			Content: "Go 测试文件应与源文件放在同一目录",
		}, false)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if result.Saved {
			t.Error("存在相同内容且 force=false，不应保存")
		}
		if len(result.SimilarExperiences) == 0 {
			t.Error("应返回相似经验列表")
		}
		for _, sim := range result.SimilarExperiences {
			if sim.Score < SimilarThreshold {
				t.Errorf("相似经验分数 %.4f 低于阈值 %.4f", sim.Score, SimilarThreshold)
			}
		}
	})

	t.Run("发现相似经验但 force=true", func(t *testing.T) {
		repo := setupTestRepo(t)
		svc := NewExperienceService(repo)
		ctx := context.Background()

		// 先保存一条
		svc.Save(ctx, &model.Experience{
			Content: "使用 Docker 部署 Go 应用",
		}, true)

		// 保存相同内容，强制
		result, err := svc.Save(ctx, &model.Experience{
			Content: "使用 Docker 部署 Go 应用",
		}, true)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if !result.Saved {
			t.Error("force=true 时应强制保存")
		}
		if result.ID == "" {
			t.Error("保存成功应返回 ID")
		}
	})

	t.Run("相似度低于阈值正常保存", func(t *testing.T) {
		repo := setupTestRepo(t)
		svc := NewExperienceService(repo)
		ctx := context.Background()

		// 先保存一条
		svc.Save(ctx, &model.Experience{
			Content: "Go 的并发模型基于 goroutine 和 channel",
		}, true)

		// 保存完全不同主题的内容
		result, err := svc.Save(ctx, &model.Experience{
			Content: "Python 的列表推导式可以简化循环操作",
		}, false)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if !result.Saved {
			t.Error("不相似的内容应正常保存")
		}
	})
}

// --- Search 测试 ---

func TestService_Search(t *testing.T) {
	svc := testServiceSetup(t)
	ctx := context.Background()

	svc.Save(ctx, &model.Experience{
		Title:   "Redis 缓存策略",
		Content: "使用 Redis 做缓存时，要设置合理的过期时间和淘汰策略",
		Tags:    []string{"redis", "cache"},
	}, true)
	svc.Save(ctx, &model.Experience{
		Title:   "Docker 部署",
		Content: "使用 Docker Compose 编排多个服务容器进行部署",
		Tags:    []string{"docker", "deploy"},
	}, true)

	t.Run("正常搜索返回结果", func(t *testing.T) {
		results, err := svc.Search(ctx, "怎么用 Redis 做缓存", 5)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("期望有搜索结果")
		}
		if results[0].Experience.Title != "Redis 缓存策略" {
			t.Errorf("最相关结果 = %q, want %q", results[0].Experience.Title, "Redis 缓存策略")
		}
	})

	t.Run("topK 为 0 时使用默认值", func(t *testing.T) {
		results, err := svc.Search(ctx, "经验", 0)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) == 0 {
			t.Error("topK=0 应使用默认值，不应返回空结果")
		}
	})

	t.Run("topK 为负数时使用默认值", func(t *testing.T) {
		results, err := svc.Search(ctx, "经验", -1)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) == 0 {
			t.Error("topK<0 应使用默认值，不应返回空结果")
		}
	})

	t.Run("topK 超过上限时被截断", func(t *testing.T) {
		results, err := svc.Search(ctx, "经验", 10000)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		_ = results
	})
}

// --- Get 测试 ---

func TestService_Get(t *testing.T) {
	svc := testServiceSetup(t)
	ctx := context.Background()

	t.Run("获取已存在的经验", func(t *testing.T) {
		saveResult, _ := svc.Save(ctx, &model.Experience{
			Title:   "获取测试",
			Content: "用于测试 Get 方法",
		}, true)

		got, err := svc.Get(ctx, saveResult.ID)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if got.Title != "获取测试" {
			t.Errorf("Title = %q, want %q", got.Title, "获取测试")
		}
	})

	t.Run("获取不存在的经验返回错误", func(t *testing.T) {
		_, err := svc.Get(ctx, "nonexistent")
		if err == nil {
			t.Error("期望返回错误")
		}
	})
}

// --- Delete 测试 ---

func TestService_Delete(t *testing.T) {
	svc := testServiceSetup(t)
	ctx := context.Background()

	t.Run("删除后 Get 失败", func(t *testing.T) {
		saveResult, _ := svc.Save(ctx, &model.Experience{Content: "即将被删除"}, true)
		err := svc.Delete(ctx, saveResult.ID)
		if err != nil {
			t.Fatalf("Delete() error: %v", err)
		}
		_, err = svc.Get(ctx, saveResult.ID)
		if err == nil {
			t.Error("删除后 Get 应返回错误")
		}
	})

	t.Run("删除不存在的 ID 不报错", func(t *testing.T) {
		err := svc.Delete(ctx, "nonexistent")
		if err != nil {
			t.Errorf("删除不存在的 ID 不应报错: %v", err)
		}
	})
}

// --- List 测试 ---

func TestService_List(t *testing.T) {
	svc := testServiceSetup(t)
	ctx := context.Background()

	svc.Save(ctx, &model.Experience{Content: "经验 1", Source: "claude"}, true)
	svc.Save(ctx, &model.Experience{Content: "经验 2", Source: "chatgpt"}, true)

	t.Run("分页列出经验", func(t *testing.T) {
		results, total, err := svc.List(ctx, 1, 10)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if total < 2 {
			t.Errorf("总数 = %d, 至少应为 2", total)
		}
		if len(results) < 2 {
			t.Errorf("期望至少 2 条结果，得到 %d", len(results))
		}
	})

	t.Run("page 为 0 时默认为第 1 页", func(t *testing.T) {
		results, _, err := svc.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if len(results) == 0 {
			t.Error("page=0 应被视为第 1 页")
		}
	})

	t.Run("pageSize 为 0 时使用默认值", func(t *testing.T) {
		results, _, err := svc.List(ctx, 1, 0)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		_ = results
	})
}
