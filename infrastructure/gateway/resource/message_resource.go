package resource

import "strconv"

type SlackMessage int64

const (
	SlackMessageHealthCheck SlackMessage = iota
	SlackMessageBuyLevaSellInvaSubRuleMet
	SlackMessageBuyLevaSellInvaSubRuleUnmetButRising
	SlackMessageBuyLevaSellInva
	SlackMessageBuyInvaSellLevaSubRuleMet
	SlackMessageBuyInvaSellLevaSubRuleUnmetButFalling
	SlackMessageBuyInvaSellLeva
	SlackMessageBuyLevaSellInvaDisappear
	SlackMessageBuyInvaSellLevaDisappear
)

var slackBotMessages = map[SlackMessage]string{
	SlackMessageHealthCheck:                           "<!channel> \nHello World!!\nThis is health check",
	SlackMessageBuyLevaSellInvaSubRuleMet:             "<!channel> *逆神注意!* \n*二刀流 買い/売り シグナル点灯/補助ルール充足* レバレッジ・インデックス(1570)を購入して、ダブルインバース・インデックス(1357)を売るシグナルがでました。\n取引を実行してください。",
	SlackMessageBuyLevaSellInvaSubRuleUnmetButRising:  "<!channel> *逆神注意!* \n*二刀流 買い/売り シグナル点灯/上昇相場* レバレッジ・インデックス(1570)を購入して、ダブルインバース・インデックス(1357)を売るシグナルがでました。\n上昇相場であるため、取引を実行してください",
	SlackMessageBuyLevaSellInva:                       "<!channel> *逆神注意!* \n*二刀流 買い/売り シグナル点灯* レバレッジ・インデックス(1570)を購入して、ダブルインバース・インデックス(1357)を売るシグナルがでました。\nダマシを忌避したい場合、補助ルールが充足した通知がされるまで、取引を行わないでください。",
	SlackMessageBuyLevaSellInvaDisappear:              "<!channel> *逆神注意!* \n*二刀流 買い/売り シグナル消滅* レバレッジ・インデックス(1570)を購入して、ダブルインバース・インデックス(1357)を売るシグナルが消失しました。",
	SlackMessageBuyInvaSellLevaSubRuleMet:             "<!channel> *逆神注意!* \n*二刀流 売り/買い シグナル点灯/補助ルール充足* ダブルインバース・インデックス(1357)を購入して、レバレッジ・インデックス(1570)を売るシグナルが出ました。\n取引を実行してください。",
	SlackMessageBuyInvaSellLevaSubRuleUnmetButFalling: "<!channel> *逆神注意!* \n*二刀流 売り/買い シグナル点灯/下降相場* ダブルインバース・インデックス(1357)を購入して、レバレッジ・インデックス(1570)を売るシグナルが出ました。\n下降相場であるため、取引を実行してください。",
	SlackMessageBuyInvaSellLeva:                       "<!channel> *逆神注意!* \n*二刀流 売り/買い シグナル点灯* ダブルインバース・インデックス(1357)を購入して、レバレッジ・インデックス(1570)を売るシグナルが出ました。\nダマシを忌避したい場合、補助ルールが充足した通知がされるまで、取引を行わないでください。",
	SlackMessageBuyInvaSellLevaDisappear:              "<!channel> *逆神注意!* \n*二刀流 売り/買い シグナル消滅* ダブルインバース・インデックス(1357)を購入して、レバレッジ・インデックス(1570)を売るシグナルが消失しました。",
}

func (s SlackMessage) String() string {
	return strconv.FormatInt(int64(s), 10)

}

// slack通知メッセージの取得
func (s SlackMessage) GetMessage() string {
	return slackBotMessages[s]
}
