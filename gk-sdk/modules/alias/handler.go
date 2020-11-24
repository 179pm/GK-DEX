package alias

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/alias/internal/types"
	dex "github.com/coinexchain/cet-sdk/types"
)

func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case types.MsgAliasUpdate:
			return handleMsgAliasUpdate(ctx, k, msg)
		default:
			return dex.ErrUnknownRequest(ModuleName, msg)
		}
	}
}

func handleMsgAliasUpdate(ctx sdk.Context, k Keeper, msg types.MsgAliasUpdate) sdk.Result {
	if msg.IsAdd {
		if err := handleAliasAdd(ctx, k, msg); err != nil {
			return err.Result()
		}
	} else {
		if err := handleAliasRemove(ctx, k, msg); err != nil {
			return err.Result()
		}
	}

	emitEvents(ctx, msg)

	return sdk.Result{
		Codespace: types.CodeSpaceAlias,
		Events:    ctx.EventManager().Events(),
	}
}

func emitEvents(ctx sdk.Context, msg types.MsgAliasUpdate) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Owner.String()),
		),
	)
	if msg.IsAdd {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeAddAlias,
				sdk.NewAttribute(types.AttributeKeyAlias, msg.Alias),
				sdk.NewAttribute(types.AttributeKeyAsDefault, fmt.Sprintf("%t", msg.AsDefault)),
			),
		)
	} else {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeRemoveAlias,
				sdk.NewAttribute(types.AttributeKeyAlias, msg.Alias),
			),
		)
	}
}

func handleAliasAdd(ctx sdk.Context, k Keeper, msg types.MsgAliasUpdate) sdk.Error {
	if types.IsOnlyForCoinEx(msg.Alias) && !k.IsTokenIssuer(ctx, dex.CET, msg.Owner) {
		return types.ErrCanOnlyBeUsedByCetOwner(msg.Alias)
	}

	addr, asDefault := k.GetAddressFromAlias(ctx, msg.Alias)
	if len(addr) != 0 && (!bytes.Equal(addr, msg.Owner) || asDefault == msg.AsDefault) {
		return types.ErrAliasAlreadyExists()
	}

	params := k.GetParams(ctx)
	ok, addNewAlias := k.AddAlias(ctx, msg.Alias, msg.Owner, msg.AsDefault, params.MaxAliasCount)
	if !ok {
		return types.ErrMaxAliasCountReached()
	}

	if addNewAlias {
		fee := params.GetFeeForAlias(msg.Alias)
		if err := k.DeductInt64CetFee(ctx, msg.Owner, fee); err != nil {
			return err
		}
	}

	return nil
}

func handleAliasRemove(ctx sdk.Context, k Keeper, msg types.MsgAliasUpdate) sdk.Error {
	addr, _ := k.GetAddressFromAlias(ctx, msg.Alias)
	if !bytes.Equal(addr, msg.Owner) {
		return types.ErrNoSuchAlias()
	}

	k.RemoveAlias(ctx, msg.Alias, msg.Owner)
	return nil
}
