package c2ssa

import (
	"fmt"
	"strconv"
	"strings"

	cparser "github.com/yaklang/yaklang/common/yak/antlr4c/parser"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

func (b *astbuilder) buildExpression(ast *cparser.ExpressionContext, isLeft bool) (ssa.Value, *ssa.Variable) {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()
	var ret ssa.Value

	getVariable := func(single *cparser.ExpressionContext, i int) *ssa.Variable {
		if s := single.Expression(i); s != nil {
			_, leftv := b.buildExpression(s.(*cparser.ExpressionContext), true)
			return leftv
		} else {
			b.NewError(ssa.Error, TAG, "can't get expression")
			return b.CreateVariable("")
		}
	}

	handlerJumpExpression := func(cond func(string) ssa.Value, trueExpr, falseExpr func() ssa.Value, name string) ssa.Value {
		id := name
		variable := b.CreateVariable(id)
		b.AssignVariable(variable, b.EmitValueOnlyDeclare(id))

		ifb := b.CreateIfBuilder()
		ifb.AppendItem(
			func() ssa.Value {
				return cond(id)
			},
			func() {
				v := trueExpr()
				variable := b.CreateVariable(id)
				b.AssignVariable(variable, v)
			},
		)
		ifb.SetElse(func() {
			v := falseExpr()
			variable := b.CreateVariable(id)
			b.AssignVariable(variable, v)
		})
		ifb.Build()
		// generator phi instruction
		v := b.ReadValue(id)
		v.SetName(ast.GetText())
		return v
	}

	// 1. 一元运算符: unary_op = (Plus | Minus | Not | Caret | Star | And) expression
	if ast.GetUnary_op() != nil && ast.Expression(0) != nil {
		op := ast.GetUnary_op().GetText()
		expr, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		if expr != nil {
			switch op {
			case "+":
				return b.EmitUnOp(ssa.OpPlus, expr), nil
			case "-":
				return b.EmitUnOp(ssa.OpNeg, expr), nil
			case "!":
				return b.EmitUnOp(ssa.OpNot, expr), nil
			case "~":
				return b.EmitUnOp(ssa.OpBitwiseNot, expr), nil
			case "*":
				if expr.GetType().GetTypeKind() == ssa.PointerKind {
					return b.GetOriginValue(expr), nil
				}
			case "&":
				if op1Var := getVariable(ast.Expression(0).(*cparser.ExpressionContext), 0); op1Var != nil {
					return b.EmitConstPointer(op1Var), nil
				}
			}
		}
		return expr, nil
	}

	// 2. 乘法/除法/取模/位移/按位与: expression mul_op = (Star | Div | Mod | LeftShift | RightShift | And) expression
	if ast.GetMul_op() != nil && len(ast.AllExpression()) >= 2 {
		op := ast.GetMul_op().GetText()
		left, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		right, _ := b.buildExpression(ast.Expression(1).(*cparser.ExpressionContext), false)
		if left != nil && right != nil {
			switch op {
			case "*":
				return b.EmitBinOp(ssa.OpMul, left, right), nil
			case "/":
				return b.EmitBinOp(ssa.OpDiv, left, right), nil
			case "%":
				return b.EmitBinOp(ssa.OpMod, left, right), nil
			case "<<":
				return b.EmitBinOp(ssa.OpShl, left, right), nil
			case ">>":
				return b.EmitBinOp(ssa.OpShr, left, right), nil
			case "&":
				return b.EmitBinOp(ssa.OpAnd, left, right), nil
			}
		}
		return left, nil
	}

	// 3. 加法/减法/按位或/按位异或: expression add_op = (Plus | Minus | Or | Caret) expression
	if ast.GetAdd_op() != nil && len(ast.AllExpression()) >= 2 {
		op := ast.GetAdd_op().GetText()
		left, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		right, _ := b.buildExpression(ast.Expression(1).(*cparser.ExpressionContext), false)
		if left != nil && right != nil {
			switch op {
			case "+":
				return b.EmitBinOp(ssa.OpAdd, left, right), nil
			case "-":
				return b.EmitBinOp(ssa.OpSub, left, right), nil
			case "|":
				return b.EmitBinOp(ssa.OpOr, left, right), nil
			case "^":
				return b.EmitBinOp(ssa.OpXor, left, right), nil
			}
		}
		return left, nil
	}

	// 4. 关系运算符: expression rel_op = (Equal | NotEqual | Less | LessEqual | Greater | GreaterEqual) expression
	if ast.GetRel_op() != nil && len(ast.AllExpression()) >= 2 {
		op := ast.GetRel_op().GetText()
		left, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		right, _ := b.buildExpression(ast.Expression(1).(*cparser.ExpressionContext), false)
		if left != nil && right != nil {
			switch op {
			case "==":
				return b.EmitBinOp(ssa.OpEq, left, right), nil
			case "!=":
				return b.EmitBinOp(ssa.OpNotEq, left, right), nil
			case "<":
				return b.EmitBinOp(ssa.OpLt, left, right), nil
			case "<=":
				return b.EmitBinOp(ssa.OpLtEq, left, right), nil
			case ">":
				return b.EmitBinOp(ssa.OpGt, left, right), nil
			case ">=":
				return b.EmitBinOp(ssa.OpGtEq, left, right), nil
			}
		}
		return left, nil
	}

	// 5. 逻辑与: expression AndAnd expression
	if ast.AndAnd() != nil && len(ast.AllExpression()) >= 2 {
		left, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		right, _ := b.buildExpression(ast.Expression(1).(*cparser.ExpressionContext), false)
		if left != nil && right != nil {
			return b.EmitBinOp(ssa.OpLogicAnd, left, right), nil
		}
		return left, nil
	}

	// 6. 逻辑或: expression OrOr expression
	if ast.OrOr() != nil && len(ast.AllExpression()) >= 2 {
		left, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		right, _ := b.buildExpression(ast.Expression(1).(*cparser.ExpressionContext), false)
		if left != nil && right != nil {
			return b.EmitBinOp(ssa.OpLogicOr, left, right), nil
		}
		return left, nil
	}

	// 7. 括号表达式: '(' expression ')'
	if ast.LeftParen() != nil && ast.Expression(0) != nil && ast.RightParen() != nil {
		return b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
	}

	// 8. 三元表达式: expression ('?' expression ':' expression)
	if ast.Question() != nil {
		condition, _ := b.buildExpression(ast.Expression(0).(*cparser.ExpressionContext), false)
		value1, _ := b.buildExpression(ast.Expression(1).(*cparser.ExpressionContext), false)
		value2, _ := b.buildExpression(ast.Expression(2).(*cparser.ExpressionContext), false)
		return handlerJumpExpression(
			func(id string) ssa.Value {
				return condition
			},
			func() ssa.Value {
				return value1
			},
			func() ssa.Value {
				return value2
			},
			ssa.AndExpressionVariable,
		), nil
	}

	// 9. 基本表达式: primaryExpression
	if p := ast.PrimaryExpression(); p != nil {
		ret, _ = b.buildPrimaryExpression(p.(*cparser.PrimaryExpressionContext), false)
		return ret, nil
	}

	// 10. 赋值表达式: assignmentExpression
	if a := ast.AssignmentExpression(); a != nil {
		return b.buildAssignmentExpression(a.(*cparser.AssignmentExpressionContext)), nil
	}

	// 11. 语句表达式: statementsExpression
	if s := ast.StatementsExpression(); s != nil {
		return b.buildStatementsExpression(s.(*cparser.StatementsExpressionContext)), nil
	}

	// 12. 类型转换表达式: castExpression
	if c := ast.CastExpression(); c != nil {
		return b.buildCastExpression(c.(*cparser.CastExpressionContext)), nil
	}

	return b.EmitConstInst(0), b.CreateVariable("")
}

func (b *astbuilder) buildAssignmentExpression(ast *cparser.AssignmentExpressionContext) ssa.Value {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	var right, newRight ssa.Value
	var left *ssa.Variable

	getValue := func() {
		if u := ast.UnaryExpression(); u != nil {
			right, _ = b.buildUnaryExpression(u.(*cparser.UnaryExpressionContext), false)
		} else if p := ast.PostfixExpression(); p != nil {
			right, _ = b.buildPostfixExpression(p.(*cparser.PostfixExpressionContext), false)
		} else if d := ast.DigitSequence(); d != nil {
			// TODO
		}
	}
	getVariable := func() {
		if u := ast.UnaryExpression(); u != nil {
			_, left = b.buildUnaryExpression(u.(*cparser.UnaryExpressionContext), true)
		} else if p := ast.PostfixExpression(); p != nil {
			_, left = b.buildPostfixExpression(p.(*cparser.PostfixExpressionContext), true)
		} else if d := ast.DigitSequence(); d != nil {
			// TODO
		}
	}

	if a := ast.AssignmentOperator(); a != nil {
		if left == nil {
			getVariable()
		}
		if right == nil {
			getValue()
		}
		if e := ast.Initializer(); e != nil {
			newRight = b.buildInitializer(e.(*cparser.InitializerContext))
		}
		if u := ast.UnaryExpression(); u != nil {
			right, _ = b.buildUnaryExpression(u.(*cparser.UnaryExpressionContext), false)
		}
		op := a.(*cparser.AssignmentOperatorContext).GetText()
		switch op {
		case "=":
			right = newRight
		case "*=":
			right = b.EmitBinOp(ssa.OpMul, right, newRight)
		case "/=":
			right = b.EmitBinOp(ssa.OpDiv, right, newRight)
		case "%=":
			right = b.EmitBinOp(ssa.OpMod, right, newRight)
		case "+=":
			right = b.EmitBinOp(ssa.OpAdd, right, newRight)
		case "-=":
			right = b.EmitBinOp(ssa.OpSub, right, newRight)
		case "<<=":
			right = b.EmitBinOp(ssa.OpShl, right, newRight)
		case ">>=":
			right = b.EmitBinOp(ssa.OpShr, right, newRight)
		case "&=":
			right = b.EmitBinOp(ssa.OpAnd, right, newRight)
		case "^=":
			right = b.EmitBinOp(ssa.OpXor, right, newRight)
		case "|=":
			right = b.EmitBinOp(ssa.OpOr, right, newRight)
		}
		b.AssignVariable(left, right)
		right.SetType(newRight.GetType())
	}

	if right == nil {
		getValue()
	}
	return right
}

func (b *astbuilder) buildUnaryExpression(ast *cparser.UnaryExpressionContext, isLeft bool) (ssa.Value, *ssa.Variable) {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()
	var right ssa.Value
	var left *ssa.Variable

	// 1. postfixExpression
	if p := ast.PostfixExpression(); p != nil {
		right, left = b.buildPostfixExpression(p.(*cparser.PostfixExpressionContext), isLeft)
	}

	// 2. unaryOperator castExpression
	if uo := ast.UnaryOperator(); uo != nil && ast.CastExpression() != nil {
		op := uo.GetText()
		var expr ssa.Value
		if c := ast.CastExpression(); c != nil {
			expr = b.buildCastExpression(c.(*cparser.CastExpressionContext))
		}

		switch op {
		case "+":
			return b.EmitUnOp(ssa.OpPlus, expr), nil
		case "-":
			return b.EmitUnOp(ssa.OpNeg, expr), nil
		case "!":
			return b.EmitUnOp(ssa.OpNot, expr), nil
		case "~":
			return b.EmitUnOp(ssa.OpBitwiseNot, expr), nil
		case "*":
			if expr.GetType().GetTypeKind() == ssa.PointerKind {
				return b.GetOriginValue(expr), nil
			}
		case "&":
			// TODO
			return b.EmitConstPointer(expr.GetLastVariable()), nil
		}

		return nil, nil
	}

	// 3. ('sizeof' | '_Alignof') '(' typeName ')'
	if (ast.AllSizeof() != nil || ast.Alignof() != nil) && ast.TypeName() != nil {
		b.buildTypeName(ast.TypeName().(*cparser.TypeNameContext))
		return nil, nil
	}

	// 4. '&&' Identifier
	if ast.AndAnd() != nil && ast.Identifier() != nil {
		return nil, nil
	}

	// 5. 前缀 ++/--
	for i := 0; i < len(ast.AllPlusPlus()); i++ {
		right = b.EmitBinOp(ssa.OpAdd, right, b.EmitConstInst(1))
		if left == nil {
			_, left = b.buildPostfixExpression(ast.PostfixExpression().(*cparser.PostfixExpressionContext), true)
			b.AssignVariable(left, right)
		}
	}
	for i := 0; i < len(ast.AllMinusMinus()); i++ {
		right = b.EmitBinOp(ssa.OpSub, right, b.EmitConstInst(1))
		if left == nil {
			_, left = b.buildPostfixExpression(ast.PostfixExpression().(*cparser.PostfixExpressionContext), true)
			b.AssignVariable(left, right)
		}
	}

	return right, left
}

func (b *astbuilder) buildPostfixExpression(ast *cparser.PostfixExpressionContext, isLeft bool) (ssa.Value, *ssa.Variable) {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	var right ssa.Value
	var left *ssa.Variable

	// 1. primaryExpression
	if p := ast.PrimaryExpression(); p != nil {
		right, left = b.buildPrimaryExpression(p.(*cparser.PrimaryExpressionContext), isLeft)
	}

	// 2. 函数调用：postfixExpression '(' argumentExpressionList? ')'
	if right != nil {
		for i := 0; i < len(ast.AllLeftParen()); i++ {
			var args ssa.Values
			if ael := ast.ArgumentExpressionList(i); ael != nil {
				args = b.buildArgumentExpressionList(ael.(*cparser.ArgumentExpressionListContext))
			}
			b.EmitCall(b.NewCall(right, args))
		}
	}

	// 3. 类型转换：'(' typeName ')' '{' initializerList ','? '}'
	if right == nil {
		if t := ast.TypeName(); t != nil {
			ssatype := b.buildTypeName(t.(*cparser.TypeNameContext))
			_ = ssatype
			if i := ast.InitializerList(); i != nil {
				right = b.buildInitializerList(i.(*cparser.InitializerListContext))
			}
		}
	}

	// 4. 数组下标：postfixExpression '[' expression ']'
	for i := 0; i < len(ast.AllLeftBracket()); i++ {
		if e := ast.Expression(i); e != nil {
			index, _ := b.buildExpression(e.(*cparser.ExpressionContext), false)
			right = b.ReadMemberCallValue(right, index)
		}
	}

	// 5. 结构体成员：postfixExpression '.' Identifier 或 '->' Identifier
	buildDotArrow := func(i int) {
		if id := ast.Identifier(i); id != nil {
			if right != nil {
				right = b.ReadMemberCallValue(right, b.EmitConstInst(id.GetText()))
			}
			if left != nil {
				left = b.CreateMemberCallVariable(left.GetValue(), b.EmitConstInst(id.GetText()))
			}
		}
	}
	for i := 0; i < len(ast.AllDot()); i++ {
		buildDotArrow(i)
	}
	for i := 0; i < len(ast.AllArrow()); i++ {
		buildDotArrow(i)
	}

	// 6. 后缀 ++/--
	for i := 0; i < len(ast.AllPlusPlus()); i++ {
		right = b.EmitBinOp(ssa.OpAdd, right, b.EmitConstInst(1))
		if left == nil {
			_, left = b.buildPrimaryExpression(ast.PrimaryExpression().(*cparser.PrimaryExpressionContext), true)
			b.AssignVariable(left, right)
		}
	}
	for i := 0; i < len(ast.AllMinusMinus()); i++ {
		right = b.EmitBinOp(ssa.OpSub, right, b.EmitConstInst(1))
		if left == nil {
			_, left = b.buildPrimaryExpression(ast.PrimaryExpression().(*cparser.PrimaryExpressionContext), true)
			b.AssignVariable(left, right)
		}
	}

	return right, left
}

func (b *astbuilder) buildInitializerList(ast *cparser.InitializerListContext) ssa.Value {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	var keys, values ssa.Values
	for _, d := range ast.AllDesignation() {
		keys = append(keys, b.buildDesignation(d.(*cparser.DesignationContext))...)
	}
	for _, i := range ast.AllInitializer() {
		values = append(values, b.buildInitializer(i.(*cparser.InitializerContext)))
	}
	obj := b.InterfaceAddFieldBuild(len(values), func(i int) ssa.Value {
		if i >= len(keys) {
			return b.EmitConstInst(i)
		}
		return keys[i]
	}, func(i int) ssa.Value {
		return values[i]
	})

	return obj
}

func (b *astbuilder) buildInitializer(ast *cparser.InitializerContext) ssa.Value {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	if a := ast.Expression(); a != nil {
		value, _ := b.buildExpression(a.(*cparser.ExpressionContext), false)
		return value
	} else if i := ast.InitializerList(); i != nil {
		return b.buildInitializerList(i.(*cparser.InitializerListContext))
	}
	return b.EmitConstInst(0)
}

func (b *astbuilder) buildDesignation(ast *cparser.DesignationContext) ssa.Values {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	if d := ast.DesignatorList(); d != nil {
		return b.buildDesignatorList(d.(*cparser.DesignatorListContext))
	}
	return nil
}

func (b *astbuilder) buildDesignatorList(ast *cparser.DesignatorListContext) ssa.Values {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	var ret ssa.Values
	for _, d := range ast.AllDesignator() {
		ret = append(ret, b.buildDesignator(d.(*cparser.DesignatorContext)))
	}
	return ret
}

func (b *astbuilder) buildDesignator(ast *cparser.DesignatorContext) ssa.Value {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	if e := ast.Expression(); e != nil {
		value, _ := b.buildExpression(e.(*cparser.ExpressionContext), false)
		return value
	} else if id := ast.Identifier(); id != nil {
		return b.EmitConstInst(id.GetText())
	}
	return b.EmitConstInst(0)
}

func (b *astbuilder) buildCastExpression(ast *cparser.CastExpressionContext) ssa.Value {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	var ret ssa.Value
	if t := ast.TypeName(); t != nil {
		ssatype := b.buildTypeName(t.(*cparser.TypeNameContext))
		if c := ast.CastExpression(); c != nil {
			ret = b.buildCastExpression(c.(*cparser.CastExpressionContext))
			ret.SetType(ssatype)
		}
	} else if u := ast.UnaryExpression(); u != nil {
		ret, _ = b.buildUnaryExpression(u.(*cparser.UnaryExpressionContext), false)
	} else if d := ast.DigitSequence(); d != nil {
		// TODO
	}

	return ret
}

func (b *astbuilder) buildPrimaryExpression(ast *cparser.PrimaryExpressionContext, isLeft bool) (ssa.Value, *ssa.Variable) {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()

	// 1. Identifier
	if id := ast.Identifier(); id != nil {
		if isLeft {
			left := b.CreateVariable(id.GetText())
			return nil, left
		} else {
			text := id.GetText()
			right := b.PeekValue(text)
			if right != nil {
				return right, nil
			}
			right = b.GetFunc(text, "")
			if right.(*ssa.Function) == nil {
				b.NewError(ssa.Warn, TAG, fmt.Sprintf("not find variable %s in current scope", text))
				right = b.ReadValue(text)
			}
			return right, nil
		}
	}
	// 2. Constant
	if c := ast.Constant(); c != nil {
		text := c.GetText()

		if len(text) > 0 {
			if text[0] == '\'' || (len(text) > 1 && text[0] == 'L' && text[1] == '\'') {
				val := parseCChar(text)
				return b.EmitConstInst(val), nil
			}
			if isCIntLiteral(text) {
				val, _ := parseCInt(text)
				return b.EmitConstInst(val), nil
			}
			if isCFloatLiteral(text) {
				val, _ := parseCFloat(text)
				return b.EmitConstInst(val), nil
			}
		}
	}
	// 3. StringLiteral+
	if n := len(ast.AllStringLiteral()); n > 0 {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			lit := ast.StringLiteral(i).GetText()
			if len(lit) >= 2 && lit[0] == '"' && lit[len(lit)-1] == '"' {
				unquoted, err := strconv.Unquote(lit)
				if err == nil {
					sb.WriteString(unquoted)
				} else {
					sb.WriteString(lit[1 : len(lit)-1])
				}
			} else {
				sb.WriteString(lit)
			}
		}
		return b.EmitConstInst(sb.String()), nil
	}
	// 4. '(' expression ')'
	if ast.LeftParen() != nil && ast.Expression() != nil && ast.RightParen() != nil {
		return b.buildExpression(ast.Expression().(*cparser.ExpressionContext), isLeft)
	}
	// 5. genericSelection
	if g := ast.GenericSelection(); g != nil {

	}
	// 6. __extension__? '(' compoundStatement ')'
	if ast.Extension() != nil && ast.LeftParen() != nil && ast.CompoundStatement() != nil && ast.RightParen() != nil {
		b.buildCompoundStatement(ast.CompoundStatement().(*cparser.CompoundStatementContext))

	}
	// 7. __builtin_va_arg '(' unaryExpression ',' typeName ')'
	if ast.BuiltinVaArg() != nil {

	}
	// 8. __builtin_offsetof '(' typeName ',' unaryExpression ')'
	if ast.BuiltinOffsetof() != nil {

	}

	return b.EmitConstInst(0), b.CreateVariable("")
}

func (b *astbuilder) buildArgumentExpressionList(ast *cparser.ArgumentExpressionListContext) ssa.Values {
	recoverRange := b.SetRange(ast.BaseParserRuleContext)
	defer recoverRange()
	var ret ssa.Values
	for _, a := range ast.AllAssignmentExpression() {
		ret = append(ret, b.buildAssignmentExpression(a.(*cparser.AssignmentExpressionContext)))
	}
	return ret
}

func parseCChar(text string) int32 {
	// 简单处理：去除前后引号，处理转义
	if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
		body := text[1 : len(text)-1]
		if len(body) == 1 {
			return int32(body[0])
		}
		if body[0] == '\\' {
			// 处理转义字符
			switch body[1] {
			case 'n':
				return '\n'
			case 't':
				return '\t'
			case 'r':
				return '\r'
			case '\\':
				return '\\'
			case '\'':
				return '\''
				// 可扩展更多
			}
		}
	}
	return 0
}

func isCIntLiteral(text string) bool {
	// 简单判断：全为数字或0x/0X/0b/0B开头
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") || strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
		return true
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

func parseCInt(text string) (int64, error) {
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		return strconv.ParseInt(text[2:], 16, 64)
	}
	if strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
		return strconv.ParseInt(text[2:], 2, 64)
	}
	if strings.HasPrefix(text, "0") && len(text) > 1 {
		return strconv.ParseInt(text, 8, 64)
	}
	return strconv.ParseInt(text, 10, 64)
}

func isCFloatLiteral(text string) bool {
	return strings.Contains(text, ".") || strings.ContainsAny(text, "eE")
}

func parseCFloat(text string) (float64, error) {
	return strconv.ParseFloat(text, 64)
}
