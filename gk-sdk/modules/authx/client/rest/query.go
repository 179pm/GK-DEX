package rest

import (
	"fmt"
	"net/http"
	"encoding/hex"
	"io/ioutil"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/gorilla/mux"

	"github.com/coinexchain/cet-sdk/modules/authx/internal/types"
	"github.com/coinexchain/cosmos-utils/client/restutil"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// register REST routes
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	r.HandleFunc("/auth/accounts/{address}", QueryAccountRequestHandlerFn(cliCtx, cdc)).Methods("GET")
	r.HandleFunc("/auth/parameters", QueryParamsHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/auth/verify", QueryVerifyRequestHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/auth/sign", SignRequestHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc("/auth/signTx/{privKey}", SignTxRequestHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc("/auth/accounts/{address}/referee", setRefereeHandleFn(cdc, cliCtx)).Methods("POST")
}

// query accountREST Handler
func QueryAccountRequestHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.StoreKey, types.QueryAccountMix)
		vars := mux.Vars(r)
		acc, err := sdk.AccAddressFromBech32(vars["address"])
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		params := auth.NewQueryAccountParams(acc)

		restutil.RestQuery(cdc, cliCtx, w, r, route, &params, nil)
	}
}

// HTTP request handler to query the authx params values
func QueryParamsHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.StoreKey, types.QueryParameters)
		restutil.RestQuery(nil, cliCtx, w, r, route, nil, nil)
	}
}

//Verify if the given signature is passed or not
func QueryVerifyRequestHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pubKey := r.URL.Query().Get("pubKey")
		message := r.URL.Query().Get("message")
		signature := r.URL.Query().Get("signature")

		pubB, _ := hex.DecodeString(pubKey)
		
		signatureB, _ := hex.DecodeString(signature)

		var pubkeyBytes secp256k1.PubKeySecp256k1
		copy(pubkeyBytes[:], pubB)
		ok := pubkeyBytes.VerifyBytes([]byte(message), signatureB)

		result := make(map[string]string)
		result["pubKey"] = pubKey
		result["message"] = message
		result["signature"] = signature
		result["result"] = fmt.Sprintf("%t",ok)
		rest.PostProcessResponse(w, cliCtx, result)
	}
}

//Sign the given message with private key
func SignRequestHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
        //private key is encoded as hex string
		privKey := r.URL.Query().Get("privKey")
		message := r.URL.Query().Get("message")
		

		privKeyB, _ := hex.DecodeString(privKey)
		
		var priv secp256k1.PrivKeySecp256k1
		copy(priv[:], privKeyB)

		pubKey := priv.PubKey()

		signature, err := priv.Sign([]byte(message))

		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		result := make(map[string]string)
		result["privKey"] = privKey
		result["pubKey"] = hex.EncodeToString(pubKey.Bytes())
		result["message"] = message
		result["signature"] = hex.EncodeToString(signature)
		rest.PostProcessResponse(w, cliCtx, result)
	}
}

func SignTxRequestHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		//private key is encoded as hex string

		privKey := r.URL.Query().Get("privKey")
		chainId := r.URL.Query().Get("chainId")
		accountNumber := r.URL.Query().Get("accountNumber")
		sequence := r.URL.Query().Get("sequence")

		tx, err := ioutil.ReadAll(r.Body)
        if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		
		stdTx, err := readStdTxFromFile(cliCtx.Codec, tx)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		privKeyB, _ := hex.DecodeString(privKey)
		var priv secp256k1.PrivKeySecp256k1
		copy(priv[:], privKeyB)

		accountNumberI,_ := strconv.Atoi(accountNumber)
		sequenceI,_ := strconv.Atoi(sequence)
		
		msg := authtypes.StdSignMsg{
			ChainID:       chainId,
			AccountNumber: uint64(accountNumberI),
			Sequence:     uint64(sequenceI),
			Fee:           stdTx.Fee,
			Msgs:          stdTx.GetMsgs(),
			Memo:          stdTx.GetMemo(),
		}


		sigBytes, err := priv.Sign(msg.Bytes())

		signature := authtypes.StdSignature{
			PubKey:    priv.PubKey(),
			Signature: sigBytes,
		}

	
		sigs := stdTx.Signatures
		if len(sigs) == 0  {
			sigs = []authtypes.StdSignature{signature}
		} else {
			sigs = append(sigs, signature)
		}

		signedStdTx := authtypes.NewStdTx(stdTx.GetMsgs(), stdTx.Fee, sigs, stdTx.GetMemo())

		rest.PostProcessResponse(w, cliCtx, signedStdTx)
	}
}

func readStdTxFromFile(cdc *codec.Codec,tx []byte) (stdTx authtypes.StdTx, err error) {
	if err = cdc.UnmarshalJSON(tx, &stdTx); err != nil {
		return
	}

	return
}


