package aibp

import (
	_ "embed"
	"github.com/yaklang/yaklang/common/ai/aid/aitool"
	"github.com/yaklang/yaklang/common/aiforge"
	"github.com/yaklang/yaklang/common/log"
)

//go:embed sf_desc_completion_prompts/desc_init.txt
var sf_desc_completion_prompt string

//go:embed sf_desc_completion_prompts/alert_init.txt
var sf_alert_completion_prompt string

func init() {
	err := aiforge.RegisterLiteForge("sf_desc_completion",
		aiforge.WithLiteForge_Prompt(sf_desc_completion_prompt),
		aiforge.WithLiteForge_OutputSchema(
			aitool.WithStringParam("title", aitool.WithParam_Required(true), aitool.WithParam_Description("规则英文标题")),
			aitool.WithStringParam("title_zh", aitool.WithParam_Required(true), aitool.WithParam_Description("规则中文标题")),
			aitool.WithStringParam("desc", aitool.WithParam_Required(true), aitool.WithParam_Description("规则描述")),
			aitool.WithStringParam("solution", aitool.WithParam_Required(true), aitool.WithParam_Description("漏洞修复方式或安全建议")),
			aitool.WithStringParam("reference", aitool.WithParam_Required(true), aitool.WithParam_Description("参考链接或文档")),
			aitool.WithNumberParam("cwe", aitool.WithParam_Description("CWE编号"), aitool.WithParam_Min(1), aitool.WithParam_Max(2000)),
		))
	if err != nil {
		log.Errorf("register freestyle chat completion failed: %v", err)
		return
	}
	err = aiforge.RegisterLiteForge(`sf_alert_completion`, aiforge.WithLiteForge_Prompt(sf_alert_completion_prompt),
		aiforge.WithLiteForge_OutputSchema(
			aitool.WithStructArrayParam("alert", []aitool.PropertyOption{
				aitool.WithParam_Description("嵌套数组结构参数"),
				aitool.WithParam_Required(),
			}, nil),
		))
	if err != nil {
		log.Errorf("register sf_alert_completion failed: %v", err)
		return
	}
}
