// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

package expression

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/privilege"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/plancodec"
	"github.com/pingcap/tidb/util/printer"
	"github.com/pingcap/tipb/go-tipb"
)

var (
	_ functionClass = &databaseFunctionClass{}
	_ functionClass = &foundRowsFunctionClass{}
	_ functionClass = &currentUserFunctionClass{}
	_ functionClass = &currentRoleFunctionClass{}
	_ functionClass = &userFunctionClass{}
	_ functionClass = &connectionIDFunctionClass{}
	_ functionClass = &lastInsertIDFunctionClass{}
	_ functionClass = &versionFunctionClass{}
	_ functionClass = &benchmarkFunctionClass{}
	_ functionClass = &charsetFunctionClass{}
	_ functionClass = &coercibilityFunctionClass{}
	_ functionClass = &collationFunctionClass{}
	_ functionClass = &rowCountFunctionClass{}
	_ functionClass = &tidbVersionFunctionClass{}
	_ functionClass = &tidbIsDDLOwnerFunctionClass{}
	_ functionClass = &tidbDecodePlanFunctionClass{}
	_ functionClass = &tidbDecodeKeyFunctionClass{}
	_ functionClass = &tidbDecodeSQLDigestsFunctionClass{}
	_ functionClass = &nextValFunctionClass{}
	_ functionClass = &lastValFunctionClass{}
	_ functionClass = &setValFunctionClass{}
	_ functionClass = &formatBytesFunctionClass{}
	_ functionClass = &formatNanoTimeFunctionClass{}
)

var (
	_ builtinFunc = &builtinDatabaseSig{}
	_ builtinFunc = &builtinFoundRowsSig{}
	_ builtinFunc = &builtinCurrentUserSig{}
	_ builtinFunc = &builtinUserSig{}
	_ builtinFunc = &builtinConnectionIDSig{}
	_ builtinFunc = &builtinLastInsertIDSig{}
	_ builtinFunc = &builtinLastInsertIDWithIDSig{}
	_ builtinFunc = &builtinVersionSig{}
	_ builtinFunc = &builtinTiDBVersionSig{}
	_ builtinFunc = &builtinRowCountSig{}
	_ builtinFunc = &builtinTiDBDecodeKeySig{}
	_ builtinFunc = &builtinTiDBDecodeSQLDigestsSig{}
	_ builtinFunc = &builtinNextValSig{}
	_ builtinFunc = &builtinLastValSig{}
	_ builtinFunc = &builtinSetValSig{}
	_ builtinFunc = &builtinFormatBytesSig{}
	_ builtinFunc = &builtinFormatNanoTimeSig{}
)

type databaseFunctionClass struct {
	baseFunctionClass
}

func (c *databaseFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Flen = 64
	sig := &builtinDatabaseSig{bf}
	return sig, nil
}

type builtinDatabaseSig struct {
	baseBuiltinFunc
}

func (b *builtinDatabaseSig) Clone() builtinFunc {
	newSig := &builtinDatabaseSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalString evals a builtinDatabaseSig.
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html
func (b *builtinDatabaseSig) evalString(row chunk.Row) (string, bool, error) {
	currentDB := b.ctx.GetSessionVars().CurrentDB
	return currentDB, currentDB == "", nil
}

type foundRowsFunctionClass struct {
	baseFunctionClass
}

func (c *foundRowsFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt)
	if err != nil {
		return nil, err
	}
	bf.tp.Flag |= mysql.UnsignedFlag
	sig := &builtinFoundRowsSig{bf}
	return sig, nil
}

type builtinFoundRowsSig struct {
	baseBuiltinFunc
}

func (b *builtinFoundRowsSig) Clone() builtinFunc {
	newSig := &builtinFoundRowsSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals a builtinFoundRowsSig.
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_found-rows
// TODO: SQL_CALC_FOUND_ROWS and LIMIT not support for now, We will finish in another PR.
func (b *builtinFoundRowsSig) evalInt(row chunk.Row) (int64, bool, error) {
	data := b.ctx.GetSessionVars()
	if data == nil {
		return 0, true, errors.Errorf("Missing session variable when eval builtin")
	}
	return int64(data.LastFoundRows), false, nil
}

type currentUserFunctionClass struct {
	baseFunctionClass
}

func (c *currentUserFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Flen = 64
	sig := &builtinCurrentUserSig{bf}
	return sig, nil
}

type builtinCurrentUserSig struct {
	baseBuiltinFunc
}

func (b *builtinCurrentUserSig) Clone() builtinFunc {
	newSig := &builtinCurrentUserSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalString evals a builtinCurrentUserSig.
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_current-user
func (b *builtinCurrentUserSig) evalString(row chunk.Row) (string, bool, error) {
	data := b.ctx.GetSessionVars()
	if data == nil || data.User == nil {
		return "", true, errors.Errorf("Missing session variable when eval builtin")
	}
	return data.User.String(), false, nil
}

type currentRoleFunctionClass struct {
	baseFunctionClass
}

func (c *currentRoleFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Flen = 64
	sig := &builtinCurrentRoleSig{bf}
	return sig, nil
}

type builtinCurrentRoleSig struct {
	baseBuiltinFunc
}

func (b *builtinCurrentRoleSig) Clone() builtinFunc {
	newSig := &builtinCurrentRoleSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalString evals a builtinCurrentUserSig.
// See https://dev.mysql.com/doc/refman/8.0/en/information-functions.html#function_current-role
func (b *builtinCurrentRoleSig) evalString(row chunk.Row) (res string, isNull bool, err error) {
	data := b.ctx.GetSessionVars()
	if data == nil || data.ActiveRoles == nil {
		return "", true, errors.Errorf("Missing session variable when eval builtin")
	}
	if len(data.ActiveRoles) == 0 {
		return "NONE", false, nil
	}
	sortedRes := make([]string, 0, 10)
	for _, r := range data.ActiveRoles {
		sortedRes = append(sortedRes, r.String())
	}
	sort.Strings(sortedRes)
	for i, r := range sortedRes {
		res += r
		if i != len(data.ActiveRoles)-1 {
			res += ","
		}
	}
	return res, false, nil
}

type userFunctionClass struct {
	baseFunctionClass
}

func (c *userFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Flen = 64
	sig := &builtinUserSig{bf}
	return sig, nil
}

type builtinUserSig struct {
	baseBuiltinFunc
}

func (b *builtinUserSig) Clone() builtinFunc {
	newSig := &builtinUserSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalString evals a builtinUserSig.
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_user
func (b *builtinUserSig) evalString(row chunk.Row) (string, bool, error) {
	data := b.ctx.GetSessionVars()
	if data == nil || data.User == nil {
		return "", true, errors.Errorf("Missing session variable when eval builtin")
	}
	return data.User.LoginString(), false, nil
}

type connectionIDFunctionClass struct {
	baseFunctionClass
}

func (c *connectionIDFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt)
	if err != nil {
		return nil, err
	}
	bf.tp.Flag |= mysql.UnsignedFlag
	sig := &builtinConnectionIDSig{bf}
	return sig, nil
}

type builtinConnectionIDSig struct {
	baseBuiltinFunc
}

func (b *builtinConnectionIDSig) Clone() builtinFunc {
	newSig := &builtinConnectionIDSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinConnectionIDSig) evalInt(_ chunk.Row) (int64, bool, error) {
	data := b.ctx.GetSessionVars()
	if data == nil {
		return 0, true, errors.Errorf("Missing session variable `builtinConnectionIDSig.evalInt`")
	}
	return int64(data.ConnectionID), false, nil
}

type lastInsertIDFunctionClass struct {
	baseFunctionClass
}

func (c *lastInsertIDFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (sig builtinFunc, err error) {
	if err = c.verifyArgs(args); err != nil {
		return nil, err
	}

	var argsTp []types.EvalType
	if len(args) == 1 {
		argsTp = append(argsTp, types.ETInt)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt, argsTp...)
	if err != nil {
		return nil, err
	}
	bf.tp.Flag |= mysql.UnsignedFlag

	if len(args) == 1 {
		sig = &builtinLastInsertIDWithIDSig{bf}
		sig.setPbCode(tipb.ScalarFuncSig_LastInsertIDWithID)
	} else {
		sig = &builtinLastInsertIDSig{bf}
		sig.setPbCode(tipb.ScalarFuncSig_LastInsertID)
	}
	return sig, err
}

type builtinLastInsertIDSig struct {
	baseBuiltinFunc
}

func (b *builtinLastInsertIDSig) Clone() builtinFunc {
	newSig := &builtinLastInsertIDSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals LAST_INSERT_ID().
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_last-insert-id.
func (b *builtinLastInsertIDSig) evalInt(row chunk.Row) (res int64, isNull bool, err error) {
	res = int64(b.ctx.GetSessionVars().StmtCtx.PrevLastInsertID)
	return res, false, nil
}

type builtinLastInsertIDWithIDSig struct {
	baseBuiltinFunc
}

func (b *builtinLastInsertIDWithIDSig) Clone() builtinFunc {
	newSig := &builtinLastInsertIDWithIDSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals LAST_INSERT_ID(expr).
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_last-insert-id.
func (b *builtinLastInsertIDWithIDSig) evalInt(row chunk.Row) (res int64, isNull bool, err error) {
	res, isNull, err = b.args[0].EvalInt(b.ctx, row)
	if isNull || err != nil {
		return res, isNull, err
	}

	b.ctx.GetSessionVars().SetLastInsertID(uint64(res))
	return res, false, nil
}

type versionFunctionClass struct {
	baseFunctionClass
}

func (c *versionFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Flen = 64
	sig := &builtinVersionSig{bf}
	return sig, nil
}

type builtinVersionSig struct {
	baseBuiltinFunc
}

func (b *builtinVersionSig) Clone() builtinFunc {
	newSig := &builtinVersionSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalString evals a builtinVersionSig.
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_version
func (b *builtinVersionSig) evalString(row chunk.Row) (string, bool, error) {
	return mysql.ServerVersion, false, nil
}

type tidbVersionFunctionClass struct {
	baseFunctionClass
}

func (c *tidbVersionFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Flen = len(printer.GetTiDBInfo())
	sig := &builtinTiDBVersionSig{bf}
	return sig, nil
}

type builtinTiDBVersionSig struct {
	baseBuiltinFunc
}

func (b *builtinTiDBVersionSig) Clone() builtinFunc {
	newSig := &builtinTiDBVersionSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalString evals a builtinTiDBVersionSig.
// This will show git hash and build time for tidb-server.
func (b *builtinTiDBVersionSig) evalString(_ chunk.Row) (string, bool, error) {
	return printer.GetTiDBInfo(), false, nil
}

type tidbIsDDLOwnerFunctionClass struct {
	baseFunctionClass
}

func (c *tidbIsDDLOwnerFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt)
	if err != nil {
		return nil, err
	}
	sig := &builtinTiDBIsDDLOwnerSig{bf}
	return sig, nil
}

type builtinTiDBIsDDLOwnerSig struct {
	baseBuiltinFunc
}

func (b *builtinTiDBIsDDLOwnerSig) Clone() builtinFunc {
	newSig := &builtinTiDBIsDDLOwnerSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals a builtinTiDBIsDDLOwnerSig.
func (b *builtinTiDBIsDDLOwnerSig) evalInt(_ chunk.Row) (res int64, isNull bool, err error) {
	ddlOwnerChecker := b.ctx.DDLOwnerChecker()
	if ddlOwnerChecker.IsOwner() {
		res = 1
	}

	return res, false, nil
}

type benchmarkFunctionClass struct {
	baseFunctionClass
}

func (c *benchmarkFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}

	// Syntax: BENCHMARK(loop_count, expression)
	// Define with same eval type of input arg to avoid unnecessary cast function.
	sameEvalType := args[1].GetType().EvalType()
	// constLoopCount is used by VecEvalInt
	// since non-constant loop count would be different between rows, and cannot be vectorized.
	var constLoopCount int64
	con, ok := args[0].(*Constant)
	if ok && con.Value.Kind() == types.KindInt64 {
		if lc, isNull, err := con.EvalInt(ctx, chunk.Row{}); err == nil && !isNull {
			constLoopCount = lc
		}
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt, types.ETInt, sameEvalType)
	if err != nil {
		return nil, err
	}
	sig := &builtinBenchmarkSig{bf, constLoopCount}
	return sig, nil
}

type builtinBenchmarkSig struct {
	baseBuiltinFunc
	constLoopCount int64
}

func (b *builtinBenchmarkSig) Clone() builtinFunc {
	newSig := &builtinBenchmarkSig{constLoopCount: b.constLoopCount}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals a builtinBenchmarkSig. It will execute expression repeatedly count times.
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_benchmark
func (b *builtinBenchmarkSig) evalInt(row chunk.Row) (int64, bool, error) {
	// Get loop count.
	var loopCount int64
	var isNull bool
	var err error
	if b.constLoopCount > 0 {
		loopCount = b.constLoopCount
	} else {
		loopCount, isNull, err = b.args[0].EvalInt(b.ctx, row)
		if isNull || err != nil {
			return 0, isNull, err
		}
	}

	// BENCHMARK() will return NULL if loop count < 0,
	// behavior observed on MySQL 5.7.24.
	if loopCount < 0 {
		return 0, true, nil
	}

	// Eval loop count times based on arg type.
	// BENCHMARK() will pass-through the eval error,
	// behavior observed on MySQL 5.7.24.
	var i int64
	arg, ctx := b.args[1], b.ctx
	switch evalType := arg.GetType().EvalType(); evalType {
	case types.ETInt:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalInt(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	case types.ETReal:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalReal(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	case types.ETDecimal:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalDecimal(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	case types.ETString:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalString(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	case types.ETDatetime, types.ETTimestamp:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalTime(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	case types.ETDuration:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalDuration(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	case types.ETJson:
		for ; i < loopCount; i++ {
			_, isNull, err = arg.EvalJSON(ctx, row)
			if err != nil {
				return 0, isNull, err
			}
		}
	default: // Should never go into here.
		return 0, true, errors.Errorf("EvalType %v not implemented for builtin BENCHMARK()", evalType)
	}

	// Return value of BENCHMARK() is always 0.
	return 0, false, nil
}

type charsetFunctionClass struct {
	baseFunctionClass
}

func (c *charsetFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	return nil, errFunctionNotExists.GenWithStackByArgs("FUNCTION", "CHARSET")
}

type coercibilityFunctionClass struct {
	baseFunctionClass
}

func (c *coercibilityFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt, args[0].GetType().EvalType())
	if err != nil {
		return nil, err
	}
	sig := &builtinCoercibilitySig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_Unspecified)
	return sig, nil
}

type builtinCoercibilitySig struct {
	baseBuiltinFunc
}

func (c *builtinCoercibilitySig) evalInt(_ chunk.Row) (res int64, isNull bool, err error) {
	return int64(c.args[0].Coercibility()), false, nil
}

func (c *builtinCoercibilitySig) Clone() builtinFunc {
	newSig := &builtinCoercibilitySig{}
	newSig.cloneFrom(&c.baseBuiltinFunc)
	return newSig
}

type collationFunctionClass struct {
	baseFunctionClass
}

func (c *collationFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	argsTps := make([]types.EvalType, 0, len(args))
	for _, arg := range args {
		argsTps = append(argsTps, arg.GetType().EvalType())
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString, argsTps...)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinCollationSig{bf}
	return sig, nil
}

type builtinCollationSig struct {
	baseBuiltinFunc
}

func (b *builtinCollationSig) Clone() builtinFunc {
	newSig := &builtinCollationSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinCollationSig) evalString(_ chunk.Row) (string, bool, error) {
	return b.args[0].GetType().Collate, false, nil
}

type rowCountFunctionClass struct {
	baseFunctionClass
}

func (c *rowCountFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (sig builtinFunc, err error) {
	if err = c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt)
	if err != nil {
		return nil, err
	}
	sig = &builtinRowCountSig{bf}
	sig.setPbCode(tipb.ScalarFuncSig_RowCount)
	return sig, nil
}

type builtinRowCountSig struct {
	baseBuiltinFunc
}

func (b *builtinRowCountSig) Clone() builtinFunc {
	newSig := &builtinRowCountSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals ROW_COUNT().
// See https://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_row-count.
func (b *builtinRowCountSig) evalInt(_ chunk.Row) (res int64, isNull bool, err error) {
	res = b.ctx.GetSessionVars().StmtCtx.PrevAffectedRows
	return res, false, nil
}

type tidbDecodeKeyFunctionClass struct {
	baseFunctionClass
}

func (c *tidbDecodeKeyFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString, types.ETString)
	if err != nil {
		return nil, err
	}
	sig := &builtinTiDBDecodeKeySig{bf}
	return sig, nil
}

type builtinTiDBDecodeKeySig struct {
	baseBuiltinFunc
}

func (b *builtinTiDBDecodeKeySig) Clone() builtinFunc {
	newSig := &builtinTiDBDecodeKeySig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// evalInt evals a builtinTiDBIsDDLOwnerSig.
func (b *builtinTiDBDecodeKeySig) evalString(row chunk.Row) (string, bool, error) {
	s, isNull, err := b.args[0].EvalString(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, err
	}
	decode := func(ctx sessionctx.Context, s string) string { return s }
	if fn := b.ctx.Value(TiDBDecodeKeyFunctionKey); fn != nil {
		decode = fn.(func(ctx sessionctx.Context, s string) string)
	}
	return decode(b.ctx, s), false, nil
}

// TiDBDecodeKeyFunctionKeyType is used to identify the decoder function in context.
type TiDBDecodeKeyFunctionKeyType int

// String() implements Stringer.
func (k TiDBDecodeKeyFunctionKeyType) String() string {
	return "tidb_decode_key"
}

// TiDBDecodeKeyFunctionKey is used to identify the decoder function in context.
const TiDBDecodeKeyFunctionKey TiDBDecodeKeyFunctionKeyType = 0

type tidbDecodeSQLDigestsFunctionClass struct {
	baseFunctionClass
}

func (c *tidbDecodeSQLDigestsFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}

	pm := privilege.GetPrivilegeManager(ctx)
	if pm != nil && !pm.RequestVerification(ctx.GetSessionVars().ActiveRoles, "", "", "", mysql.ProcessPriv) {
		return nil, errSpecificAccessDenied.GenWithStackByArgs("PROCESS")
	}

	var argTps []types.EvalType
	if len(args) > 1 {
		argTps = []types.EvalType{types.ETString, types.ETInt}
	} else {
		argTps = []types.EvalType{types.ETString}
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString, argTps...)
	if err != nil {
		return nil, err
	}
	sig := &builtinTiDBDecodeSQLDigestsSig{bf}
	return sig, nil
}

type builtinTiDBDecodeSQLDigestsSig struct {
	baseBuiltinFunc
}

func (b *builtinTiDBDecodeSQLDigestsSig) Clone() builtinFunc {
	newSig := &builtinTiDBDecodeSQLDigestsSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinTiDBDecodeSQLDigestsSig) evalString(row chunk.Row) (string, bool, error) {
	args := b.getArgs()
	digestsStr, isNull, err := args[0].EvalString(b.ctx, row)
	if err != nil {
		return "", true, err
	}
	if isNull {
		return "", true, nil
	}

	stmtTruncateLength := int64(0)
	if len(args) > 1 {
		stmtTruncateLength, isNull, err = args[1].EvalInt(b.ctx, row)
		if err != nil {
			return "", true, err
		}
		if isNull {
			stmtTruncateLength = 0
		}
	}

	var digests []interface{}
	err = json.Unmarshal([]byte(digestsStr), &digests)
	if err != nil {
		const errMsgMaxLength = 32
		if len(digestsStr) > errMsgMaxLength {
			digestsStr = digestsStr[:errMsgMaxLength] + "..."
		}
		b.ctx.GetSessionVars().StmtCtx.AppendWarning(errIncorrectArgs.GenWithStack("The argument can't be unmarshalled as JSON array: '%s'", digestsStr))
		return "", true, nil
	}

	// Query the SQL Statements by digests.
	retriever := NewSQLDigestTextRetriever()
	for _, item := range digests {
		if item != nil {
			digest, ok := item.(string)
			if ok {
				retriever.SQLDigestsMap[digest] = ""
			}
		}
	}

	// Querying may take some time and it takes a context.Context as argument, which is not available here.
	// We simply create a context with a timeout here.
	timeout := time.Duration(b.ctx.GetSessionVars().MaxExecutionTime) * time.Millisecond
	if timeout == 0 || timeout > 20*time.Second {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = retriever.RetrieveGlobal(ctx, b.ctx)
	if err != nil {
		if errors.Cause(err) == context.DeadlineExceeded || errors.Cause(err) == context.Canceled {
			return "", true, errUnknown.GenWithStack("Retrieving cancelled internally with error: %v", err)
		}

		b.ctx.GetSessionVars().StmtCtx.AppendWarning(errUnknown.GenWithStack("Retrieving statements information failed with error: %v", err))
		return "", true, nil
	}

	// Collect the result.
	result := make([]interface{}, len(digests))
	for i, item := range digests {
		if item == nil {
			continue
		}
		if digest, ok := item.(string); ok {
			if stmt, ok := retriever.SQLDigestsMap[digest]; ok && len(stmt) > 0 {
				// Truncate too-long statements if necessary.
				if stmtTruncateLength > 0 && int64(len(stmt)) > stmtTruncateLength {
					stmt = stmt[:stmtTruncateLength] + "..."
				}
				result[i] = stmt
			}
		}
	}

	resultStr, err := json.Marshal(result)
	if err != nil {
		b.ctx.GetSessionVars().StmtCtx.AppendWarning(errUnknown.GenWithStack("Marshalling result as JSON failed with error: %v", err))
		return "", true, nil
	}

	return string(resultStr), false, nil
}

type tidbDecodePlanFunctionClass struct {
	baseFunctionClass
}

func (c *tidbDecodePlanFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString, types.ETString)
	if err != nil {
		return nil, err
	}
	sig := &builtinTiDBDecodePlanSig{bf}
	return sig, nil
}

type builtinTiDBDecodePlanSig struct {
	baseBuiltinFunc
}

func (b *builtinTiDBDecodePlanSig) Clone() builtinFunc {
	newSig := &builtinTiDBDecodePlanSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinTiDBDecodePlanSig) evalString(row chunk.Row) (string, bool, error) {
	planString, isNull, err := b.args[0].EvalString(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, err
	}
	planTree, err := plancodec.DecodePlan(planString)
	if err != nil {
		return planString, false, nil
	}
	return planTree, false, nil
}

type nextValFunctionClass struct {
	baseFunctionClass
}

func (c *nextValFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt, types.ETString)
	if err != nil {
		return nil, err
	}
	sig := &builtinNextValSig{bf}
	bf.tp.Flen = 10
	return sig, nil
}

type builtinNextValSig struct {
	baseBuiltinFunc
}

func (b *builtinNextValSig) Clone() builtinFunc {
	newSig := &builtinNextValSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinNextValSig) evalInt(row chunk.Row) (int64, bool, error) {
	sequenceName, isNull, err := b.args[0].EvalString(b.ctx, row)
	if isNull || err != nil {
		return 0, isNull, err
	}
	db, seq := getSchemaAndSequence(sequenceName)
	if len(db) == 0 {
		db = b.ctx.GetSessionVars().CurrentDB
	}
	// Check the tableName valid.
	sequence, err := util.GetSequenceByName(b.ctx.GetInfoSchema(), model.NewCIStr(db), model.NewCIStr(seq))
	if err != nil {
		return 0, false, err
	}
	// Do the privilege check.
	checker := privilege.GetPrivilegeManager(b.ctx)
	user := b.ctx.GetSessionVars().User
	if checker != nil && !checker.RequestVerification(b.ctx.GetSessionVars().ActiveRoles, db, seq, "", mysql.InsertPriv) {
		return 0, false, errSequenceAccessDenied.GenWithStackByArgs("INSERT", user.AuthUsername, user.AuthHostname, seq)
	}
	nextVal, err := sequence.GetSequenceNextVal(b.ctx, db, seq)
	if err != nil {
		return 0, false, err
	}
	// update the sequenceState.
	b.ctx.GetSessionVars().SequenceState.UpdateState(sequence.GetSequenceID(), nextVal)
	return nextVal, false, nil
}

type lastValFunctionClass struct {
	baseFunctionClass
}

func (c *lastValFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt, types.ETString)
	if err != nil {
		return nil, err
	}
	sig := &builtinLastValSig{bf}
	bf.tp.Flen = 10
	return sig, nil
}

type builtinLastValSig struct {
	baseBuiltinFunc
}

func (b *builtinLastValSig) Clone() builtinFunc {
	newSig := &builtinLastValSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinLastValSig) evalInt(row chunk.Row) (int64, bool, error) {
	sequenceName, isNull, err := b.args[0].EvalString(b.ctx, row)
	if isNull || err != nil {
		return 0, isNull, err
	}
	db, seq := getSchemaAndSequence(sequenceName)
	if len(db) == 0 {
		db = b.ctx.GetSessionVars().CurrentDB
	}
	// Check the tableName valid.
	sequence, err := util.GetSequenceByName(b.ctx.GetInfoSchema(), model.NewCIStr(db), model.NewCIStr(seq))
	if err != nil {
		return 0, false, err
	}
	// Do the privilege check.
	checker := privilege.GetPrivilegeManager(b.ctx)
	user := b.ctx.GetSessionVars().User
	if checker != nil && !checker.RequestVerification(b.ctx.GetSessionVars().ActiveRoles, db, seq, "", mysql.SelectPriv) {
		return 0, false, errSequenceAccessDenied.GenWithStackByArgs("SELECT", user.AuthUsername, user.AuthHostname, seq)
	}
	return b.ctx.GetSessionVars().SequenceState.GetLastValue(sequence.GetSequenceID())
}

type setValFunctionClass struct {
	baseFunctionClass
}

func (c *setValFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETInt, types.ETString, types.ETInt)
	if err != nil {
		return nil, err
	}
	sig := &builtinSetValSig{bf}
	bf.tp.Flen = args[1].GetType().Flen
	return sig, nil
}

type builtinSetValSig struct {
	baseBuiltinFunc
}

func (b *builtinSetValSig) Clone() builtinFunc {
	newSig := &builtinSetValSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinSetValSig) evalInt(row chunk.Row) (int64, bool, error) {
	sequenceName, isNull, err := b.args[0].EvalString(b.ctx, row)
	if isNull || err != nil {
		return 0, isNull, err
	}
	db, seq := getSchemaAndSequence(sequenceName)
	if len(db) == 0 {
		db = b.ctx.GetSessionVars().CurrentDB
	}
	// Check the tableName valid.
	sequence, err := util.GetSequenceByName(b.ctx.GetInfoSchema(), model.NewCIStr(db), model.NewCIStr(seq))
	if err != nil {
		return 0, false, err
	}
	// Do the privilege check.
	checker := privilege.GetPrivilegeManager(b.ctx)
	user := b.ctx.GetSessionVars().User
	if checker != nil && !checker.RequestVerification(b.ctx.GetSessionVars().ActiveRoles, db, seq, "", mysql.InsertPriv) {
		return 0, false, errSequenceAccessDenied.GenWithStackByArgs("INSERT", user.AuthUsername, user.AuthHostname, seq)
	}
	setValue, isNull, err := b.args[1].EvalInt(b.ctx, row)
	if isNull || err != nil {
		return 0, isNull, err
	}
	return sequence.SetSequenceVal(b.ctx, setValue, db, seq)
}

func getSchemaAndSequence(sequenceName string) (string, string) {
	res := strings.Split(sequenceName, ".")
	if len(res) == 1 {
		return "", res[0]
	}
	return res[0], res[1]
}

type formatBytesFunctionClass struct {
	baseFunctionClass
}

func (c *formatBytesFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString, types.ETReal)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	sig := &builtinFormatBytesSig{bf}
	return sig, nil
}

type builtinFormatBytesSig struct {
	baseBuiltinFunc
}

func (b *builtinFormatBytesSig) Clone() builtinFunc {
	newSig := &builtinFormatBytesSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// formatBytes evals a builtinFormatBytesSig.
// See https://dev.mysql.com/doc/refman/8.0/en/performance-schema-functions.html#function_format-bytes
func (b *builtinFormatBytesSig) evalString(row chunk.Row) (string, bool, error) {
	val, isNull, err := b.args[0].EvalReal(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, err
	}
	return GetFormatBytes(val), false, nil
}

type formatNanoTimeFunctionClass struct {
	baseFunctionClass
}

func (c *formatNanoTimeFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString, types.ETReal)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	sig := &builtinFormatNanoTimeSig{bf}
	return sig, nil
}

type builtinFormatNanoTimeSig struct {
	baseBuiltinFunc
}

func (b *builtinFormatNanoTimeSig) Clone() builtinFunc {
	newSig := &builtinFormatNanoTimeSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

// formatNanoTime evals a builtinFormatNanoTimeSig, as time unit in TiDB is always nanosecond, not picosecond.
// See https://dev.mysql.com/doc/refman/8.0/en/performance-schema-functions.html#function_format-pico-time
func (b *builtinFormatNanoTimeSig) evalString(row chunk.Row) (string, bool, error) {
	val, isNull, err := b.args[0].EvalReal(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, err
	}
	return GetFormatNanoTime(val), false, nil
}
