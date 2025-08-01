package yakgrpc

import (
	"context"
	"github.com/yaklang/yaklang/common/yakgrpc/yakit"
	"strings"

	"github.com/yaklang/yaklang/common/ai/aid/aitool/buildinaitools/yakscripttools/metadata/genmetadata"
	"github.com/yaklang/yaklang/common/consts"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/bizhelper"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

func (s *Server) GetAIToolList(ctx context.Context, req *ypb.GetAIToolListRequest) (*ypb.GetAIToolListResponse, error) {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return nil, utils.Errorf("database not initialized")
	}

	var (
		pagination *bizhelper.Paginator
		tools      []*schema.AIYakTool
		err        error
	)

	// If ToolName is provided, search by exact name
	if req.GetToolName() != "" {
		tool, err := yakit.GetAIYakTool(db, req.GetToolName())
		if err != nil {
			return &ypb.GetAIToolListResponse{
				Tools: []*ypb.AITool{},
			}, nil
		}

		// Convert single tool to response format
		return &ypb.GetAIToolListResponse{
			Tools: []*ypb.AITool{
				{
					Name:        tool.Name,
					Description: tool.Description,
					Content:     tool.Content,
					ToolPath:    tool.Path,
					Keywords:    strings.Split(tool.Keywords, ","),
					IsFavorite:  tool.IsFavorite,
				},
			},
		}, nil
	}

	// Otherwise use Query for fuzzy search with pagination
	pagination, tools, err = yakit.SearchAIYakToolWithPagination(db, req.GetQuery(), req.GetOnlyFavorites(), req.GetPagination())
	if err != nil {
		log.Errorf("failed to search AI tools: %s", err)
		return &ypb.GetAIToolListResponse{
			Tools: []*ypb.AITool{},
		}, nil
	}

	// Convert tools to response format
	var result []*ypb.AITool
	for _, tool := range tools {
		result = append(result, &ypb.AITool{
			Name:        tool.Name,
			Description: tool.Description,
			Content:     tool.Content,
			ToolPath:    tool.Path,
			Keywords:    strings.Split(tool.Keywords, ","),
			IsFavorite:  tool.IsFavorite,
		})
	}

	// Prepare response with pagination info
	return &ypb.GetAIToolListResponse{
		Tools: result,
		Pagination: &ypb.Paging{
			Page:    int64(pagination.Page),
			Limit:   int64(pagination.Limit),
			OrderBy: req.GetPagination().GetOrderBy(),
			Order:   req.GetPagination().GetOrder(),
		},
		Total: int64(pagination.TotalRecord),
	}, nil
}

func (s *Server) AIToolGenerateMetadata(ctx context.Context, req *ypb.AIToolGenerateMetadataRequest) (*ypb.AIToolGenerateMetadataResponse, error) {
	metadata, err := genmetadata.GenerateMetadataFromCodeContent(req.GetToolName(), req.GetContent())
	if err != nil {
		return nil, utils.Errorf("failed to generate AI tool metadata: %s", err)
	}
	return &ypb.AIToolGenerateMetadataResponse{
		Name:        metadata.Name,
		Description: metadata.Description,
		Keywords:    metadata.Keywords,
	}, nil
}

func (s *Server) SaveAITool(ctx context.Context, req *ypb.SaveAIToolRequest) (*ypb.DbOperateMessage, error) {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return nil, utils.Errorf("database not initialized")
	}

	tool := &schema.AIYakTool{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Content:     req.GetContent(),
		Path:        req.GetToolPath(),
		Keywords:    strings.Join(req.GetKeywords(), ","),
	}

	affected, err := yakit.SaveAIYakTool(db, tool)
	if err != nil {
		return nil, utils.Errorf("failed to create AI tool: %s", err)
	}
	return &ypb.DbOperateMessage{
		TableName:  (&schema.AIYakTool{}).TableName(),
		Operation:  "create",
		EffectRows: affected,
	}, nil
}

func (s *Server) DeleteAITool(ctx context.Context, req *ypb.DeleteAIToolRequest) (*ypb.DbOperateMessage, error) {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return nil, utils.Errorf("database not initialized")
	}

	affected, err := yakit.DeleteAIYakTools(db, req.GetToolNames()...)
	if err != nil {
		return nil, utils.Errorf("failed to delete AI tool: %s", err)
	}
	return &ypb.DbOperateMessage{
		TableName:  (&schema.AIYakTool{}).TableName(),
		Operation:  "delete",
		EffectRows: affected,
	}, nil
}

func (s *Server) ToggleAIToolFavorite(ctx context.Context, req *ypb.ToggleAIToolFavoriteRequest) (*ypb.ToggleAIToolFavoriteResponse, error) {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return nil, utils.Errorf("database not initialized")
	}

	if req.GetToolName() == "" {
		return nil, utils.Errorf("tool name cannot be empty")
	}

	isFavorite, err := yakit.ToggleAIYakToolFavorite(db, req.GetToolName())
	if err != nil {
		return nil, utils.Errorf("failed to toggle AI tool favorite status: %s", err)
	}

	message := "Tool added to favorites"
	if !isFavorite {
		message = "Tool removed from favorites"
	}

	return &ypb.ToggleAIToolFavoriteResponse{
		IsFavorite: isFavorite,
		Message:    message,
	}, nil
}
