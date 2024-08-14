package custom_errors

import "errors"

var PLAYER_EXIST_ERROR = errors.New("player already exist")
var PLAYER_NOT_EXIST_ERROR = errors.New("player not exist")
var BIND_JSON_ERROR = errors.New("bind json error")
var AMOUNT_ERROR = errors.New("amount error")
var WITHDRAW_HANDLE_ERROR = errors.New("withdraw has already be confirmed or not handled")
var FOOD_NOT_ENOUGH_ERROR = errors.New("food not enough")
var SPEAK_NOT_ENOUGH_ERROR = errors.New("speak not enough")
var SESSION_ID_EXIST_ERROR = errors.New("session id already exist")
var UNVALUABLE_SESSION_ID_ERROR = errors.New("unvaluable session id")
var ETH_ADDRESS_EXIST_ERROR = errors.New("eth address already exist")
var GET_USERINFO_ERROR = errors.New("get userinfo error")
