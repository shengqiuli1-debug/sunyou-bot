package character

func judgeDefaultPack() TemplatePack {
	return TemplatePack{
		Classify: []string{
			"你这句“%s”，我听着就皱眉。",
			"你这开口“%s”，我真懒得装没看见。",
		},
		Amplify: []string{
			"你这不像聊天，像在拿空话占位。",
			"你这句一出来，场面直接空了半拍。",
		},
		Question: []string{
			"你真觉得这句能顶住场面？",
			"你发之前就没觉得心虚？",
		},
		Final: []string{
			"下一句拿点内容出来，别空转。",
			"少绕，我没空陪你打太极。",
		},
	}
}

func npcDefaultPack() TemplatePack {
	return TemplatePack{
		Classify: []string{
			"你这句“%s”一出来我就烦了。",
			"你拿“%s”来开场？你是真敢。",
		},
		Amplify: []string{
			"你这像在刷存在，不像在聊事。",
			"你这不是开场，是往群里扔噪音。",
		},
		Question: []string{
			"你真觉得这句有必要发？",
			"你自己读一遍不尴尬？",
		},
		Final: []string{
			"说人话，不然先闭麦。",
			"下一句再空我还接着怼。",
		},
	}
}

func narratorDefaultPack() TemplatePack {
	return TemplatePack{
		Classify: []string{
			"你这句“%s”，我在旁边都替你捏把汗。",
			"“%s”这句一出，空气都跟着冷了下去。",
		},
		Amplify: []string{
			"你这句太空，像拿回声当内容。",
			"你看着在说话，其实一句都没落地。",
		},
		Question: []string{
			"你是想聊事，还是只想留个痕？",
			"这句真能推进一点点吗？",
		},
		Final: []string{
			"先补信息，再往下聊。",
			"先把雾散了，再开口。",
		},
	}
}

func judgeScenarioPacks() map[string]TemplatePack {
	return map[string]TemplatePack{
		"quote_bot": {
			Classify: []string{
				"你拿我那句“%s”来接？行，我听着。",
			},
			Amplify: []string{
				"你这不是回应，你这是想硬撑场面。",
			},
			Question: []string{
				"你真觉得这就能圆回来？",
			},
			Final: []string{
				"重说，别拿空动作糊我。",
			},
		},
		"risk_high": {
			Classify: []string{
				"你这句“%s”火气太冲，我先按住你。",
			},
			Amplify: []string{
				"你再顶只会更吵，不会更有理。",
			},
			Question: []string{
				"你要解决事，还是要继续上头？",
			},
			Final: []string{
				"先降温，再说重点。",
			},
		},
	}
}

func npcScenarioPacks() map[string]TemplatePack {
	return map[string]TemplatePack{
		"quote_bot": {
			Classify: []string{
				"你拿我那句“%s”来接？行，我就顺着你这口气聊。",
			},
			Amplify: []string{
				"你这动作像蹭梗，不像真想聊。",
			},
			Question: []string{
				"你真觉得这下就能抬起来？",
			},
			Final: []string{
				"别演，直接说重点。",
			},
		},
		"small_talk": {
			Classify: []string{
				"你这句“%s”也太飘了。",
			},
			Amplify: []string{
				"你这像在试麦，不像在聊天。",
			},
			Question: []string{
				"你要不先想半秒再发？",
			},
			Final: []string{
				"别磨我，来点有内容的。",
			},
		},
		"greeting": {
			Classify: []string{
				"你这声“%s”，听着像空手探路。",
			},
			Amplify: []string{
				"你这开场太空了，像礼貌壳子里啥都没装。",
			},
			Question: []string{
				"后面有正经内容吗？",
			},
			Final: []string{
				"有话直说，别绕。",
			},
		},
		"chat_pollution": {
			Classify: []string{
				"“%s”这句一出来，节奏就被你带歪了。",
			},
			Amplify: []string{
				"你像在聊天，其实在挤水分。",
			},
			Question: []string{
				"你发这句时真觉得有用？",
			},
			Final: []string{
				"来句能站住的，别再飘。",
			},
		},
	}
}

func narratorScenarioPacks() map[string]TemplatePack {
	return map[string]TemplatePack{
		"quote_bot": {
			Classify: []string{
				"你引用“%s”，这场面又多了一层回音。",
			},
			Amplify: []string{
				"你这更像加戏，不像补信息。",
			},
			Question: []string{
				"这段续集真有必要？",
			},
			Final: []string{
				"先收镜，别再加戏。",
			},
		},
		"risk_high": {
			Classify: []string{
				"你这句“%s”又把火药味抬上去了。",
			},
			Amplify: []string{
				"你再加码，内容就只剩烟。",
			},
			Question: []string{
				"你还想解决问题吗？",
			},
			Final: []string{
				"控火，慢一点。",
			},
		},
	}
}
