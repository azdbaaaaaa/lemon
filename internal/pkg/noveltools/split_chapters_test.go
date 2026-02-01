package noveltools

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChapterSplitter_Split(t *testing.T) {
	Convey("ChapterSplitter.Split 能正确切分小说内容", t, func() {
		splitter := NewChapterSplitter()

		Convey("空内容应返回 nil", func() {
			result := splitter.Split("", 50)
			So(result, ShouldBeNil)
		})

		Convey("空白内容应返回 nil", func() {
			result := splitter.Split("   \n\n  ", 50)
			So(result, ShouldBeNil)
		})

		Convey("按章节标题切分（中文格式）", func() {
			content := `第一章 开始
这是第一章的内容，包含了很多文字。

第二章 发展
这是第二章的内容，继续讲述故事。

第三章 高潮
这是第三章的内容，故事达到高潮。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)
			So(result[0].Title, ShouldContainSubstring, "第一章")
			So(result[0].Text, ShouldContainSubstring, "这是第一章的内容")
			So(result[1].Title, ShouldContainSubstring, "第二章")
			So(result[1].Text, ShouldContainSubstring, "这是第二章的内容")
		})

		Convey("按章节标题切分（包含标题名称，如'第三章 怒打纨绔'）", func() {
			content := `第一章 开始
这是第一章的内容。

第二章 发展
这是第二章的内容。

第三章 怒打纨绔
这是第三章的内容，讲述了怒打纨绔的故事。

第四章 新的开始
这是第四章的内容，故事继续发展。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldEqual, 4)
			So(result[0].Title, ShouldContainSubstring, "第一章")
			So(result[1].Title, ShouldContainSubstring, "第二章")
			So(result[2].Title, ShouldContainSubstring, "第三章")
			So(result[2].Title, ShouldContainSubstring, "怒打纨绔")
			So(result[3].Title, ShouldContainSubstring, "第四章")
			So(result[2].Text, ShouldContainSubstring, "这是第三章的内容")
			So(result[3].Text, ShouldContainSubstring, "这是第四章的内容")
		})

		Convey("按章节标题切分（实际小说内容：第三章和第四章之间有大量内容）", func() {
			content := `第三章 怒打纨绔

　　就在赵硕被众人的目光盯的坐立不安的时候，夫子轻咳一声，神情怪异的看了赵硕一眼道："好了，大家静一静，我想赵硕方才是和大家开了一个玩笑，我们继续"
　　吐出一口浊气，赵硕感激的看了夫子一眼，老老实实的坐在那里，抱着多听少说的念头，尽可能的多了解一些这个世界，免得再犯一些常识性的错误，不然的话，只怕真的要被当成傻子看待了。
　　可惜的是赵硕想要从夫子这里得到更多的关于这个世界的常识有些不太可能了，因为随着夫子许多总结性的话语说出，赵硕才知道过了今天自己就该离开学馆，正式成年了。
　　迷迷糊糊的走出学堂的时候，一股清新的空气扑面而来让迷茫的赵硕清醒了许多，正准备打起精神回家的时候，忽然一股大力从背后传来，紧接着赵硕就感到自己整个人向前栽倒了过去。
　　噗通一声，随着一股剧痛传来，赵硕趴在地上，膝盖处传来火辣辣的痛意，转过身来，当看清楚站在一旁的人是谁的时候，心中不由得升起一股怒意。
　　陈天贵一脸得意的看着赵硕嘿嘿笑道："赵草包，真想不到啊，你还真是深藏不露啊，怎么一天不见就又傻了那么多啊……"
　　一旁和陈天贵站在一起的几个人闻言似乎想到了不久前赵硕的一番话不禁跟着哈哈大笑起来。
　　深吸一口气，赵硕握紧拳头，微微低着头，眼中闪过一道寒光，不过很快慢慢的将握紧的拳头松开。
　　缓缓的站起身来，忍着膝盖上传来的痛意，略带踉跄的向前走，对于陈天贵的挑衅根本不予回应。
　　"二哥，你怎么了"
　　赵鸾见到赵硕的情形，眼中闪过一道痛惜的神色，快步跑到赵硕的身边，一边搀扶着赵硕一边恶狠狠的盯着正色*的打量着她的陈天贵道："陈天贵，你竟然又欺负二哥！"
　　虽然发怒，可是清丽动人的赵鸾却显得愈发的靓丽，直看的陈天贵狂咽口水。
　　陈天贵恨不得将赵鸾给吞到肚子里去，向前几步欺到赵鸾和赵硕的身前，一脸陶醉的吸了一口空气道："嗯，真是好香啊！"
　　赵鸾脸上泛起泛起一丝晕红，眼中满是羞怒的神色，下意识的后退了一步。
　　前生身为孤儿，从没感受到多少亲情，可是短短一个早上的时间，却让赵硕从赵鸾以及楚秀那里感受到了什么叫做亲情，什么事家的感觉。
　　此刻见到陈天贵竟然如此的戏弄赵鸾，赵硕忍着痛意向前一步挡在赵鸾的身前，猛的握紧了拳头冲着陈天贵那张可恶的大脸狠狠的砸了上去，口中怒吼道："欺负我可以，但是不许欺负我妹妹！"
　　"啊！"
　　根本就没有一点防备的陈天贵哪里会想到一向懦弱的赵硕竟然会突然爆发给自己这么一下，顿时只觉得脸上火辣辣，鲜血、眼泪不由自主的流下来，脸上像是开了花似地。
　　赵硕见到陈天贵那副凄惨的模样不禁一阵快意，他娘的，老子可不是以前的那个懦夫了，竟然敢调戏老子的妹子，不打你个满脸桃花开我就不是赵硕。
　　站在陈天贵身边的几人被突如其来的变化给惊呆了，甚至有人不敢相信的揉了揉眼睛，当确信陈天贵的确被赵硕一拳给砸的满脸鲜血不由自主的退后一步。
　　见鬼，一向软弱的赵硕竟然会给陈天贵来这么一下，真是不可思议啊。
　　反应过来之后，陈天贵伸手在脸上抹了一把，鼻子像是碎了一般，眼泪哗哗的流下来，虽然是酸痛难忍，但是当看到赵硕的时候，陈天贵不禁怒由心生，仿佛是受了天大的屈辱一般，冲着赵硕吼道："好你个赵硕，你竟然敢打我，你真是找死"
　　见到陈天贵那副狰狞的模样，赵鸾吓了一跳，连忙拉着赵硕道："二哥，咱们快跑！"
　　可惜的是赵硕与赵鸾两人根本就只是普通的少年罢了，根本就没有办法同陈天贵这等进行过星图开启灵窍的人相比。
　　虽然陈天贵废物了一些，但是修行之人的力量确实不是常人可以抵抗的。
　　当拳头落在身上的时候，赵硕只觉得一股股的痛意传来，任由赵硕反抗，但是他的力气比起陈天贵来当真是差了太多，只能护住要害倒在地上任由陈天贵踢打发泄。
　　赵鸾先是吓了一跳紧接着像是发怒了的狮子一般冲着陈天贵扑上去，可惜的是她那点力气甚至连赵硕都比不上，哪里能帮到什么忙，若非陈天贵打她的主意的话，只怕没几下就要被打的不能动弹了。
　　全身的骨头仿佛断了一般，赵硕咬紧牙关愣是不发出一声来。
　　此时正是下学的时间，赵硕与陈天贵等人搞出的动静立刻就吸引了不少人围观。
　　几名身着华丽衣衫的少年男女看到陈天贵踢打赵硕的时候不禁皱了皱眉头，不过见到赵硕竟然如此的坚毅，眼中露出诧异的神色，毕竟赵硕生性懦弱他们还是有所耳闻的。
　　但是现在看陈天贵满脸鲜血的模样，如果不是陈天贵口中的怒吼声的话，他们还真的不敢相信陈天贵脸上的鲜血会是一向懦弱的赵硕给打的。
　　"够了，陈天贵，出口气也就算了，这里是学馆，同时别忘了，你是一名修行之人，如此欺负一个普通人，你不要脸面，我们还丢不起这个人呢！"
　　一名锦衣少年淡淡的道，声音虽然不大，可是听在陈天贵的耳中却是如同惊雷一般。
　　一脸艳羡和敬畏的看了那少年一眼，陈天贵狠狠地在赵硕的身上踹了一脚道："赵硕，今天算你命大，看在赵礼公子的面子上，暂且饶你一命。"
　　赵鸾扑到赵硕的身边将凄惨无比的赵硕搀扶起来，力气太小的缘故，一个踉跄，两人差点一起倒在地上。
　　透过满脸的鲜血，赵硕先是冷冷的看了一眼陈天贵，目光又在那几名身着锦衣华服的少年男女身上扫过，强自支撑着身体在一众人各异的目光中离开。
　　出了学馆，赵鸾看到赵硕满脸鲜血的模样不禁道："二哥，都怪我，你要是出了什么事情的话……"
　　赵硕颤声道："傻丫头，你是我妹妹，我是你亲哥哥，难道要二哥要眼看着你被人欺负不成"
　　赵鸾闻言，眼睛一酸，眼泪不由自主的流了下来，赵硕见了强自露出一丝笑意道："都是大姑娘了，再哭的话可就不好看了"
　　伸手将脸上的泪水抹去，赵鸾道："二哥，我带你去看大夫！"
　　当两人从医馆出来的时候，赵硕的模样已经好了许多，让两人松了一口气的是好在赵硕护住了要害部位，只是受了一些皮肉伤而已，修养一些时日就会没事了。
　　回到家中的时候，楚秀看到一双儿女那副模样差点昏过去，连忙将赵硕扶到房间之中，一脸疼惜的看着赵硕道："这是怎么了，早上出去的时候还是好好的，怎么会弄成这幅模样？"
　　******************
　　收藏，推荐，有票票要砸

第四章 父神盘古

　　赵鸾将事情的经过讲了一遍道："娘亲，都是陈天贵，如果不是他的话，二哥也不会成这幅模样"
　　见到楚秀那充满了母爱的目光，赵硕心中一暖道："娘亲，我没事的，我可是哥哥，只要我这当哥哥的还有一口气在，就绝对不允许任何人欺负小妹"
　　赵鸾闻言不禁颤声道："哥！"
　　楚秀则是一脸欣慰和赞赏的看着赵硕道："好，硕儿说的对，只要你们兄妹和睦，娘亲就什么都不祈求了"皱了皱眉头，赵硕道："可惜的是这次招惹了陈天贵，只怕陈天贵对小妹野心不死啊"
　　楚秀闻言，眸中寒光一闪，沉吟一番，看向赵鸾，赵鸾轻轻一笑道："二哥，娘亲，你们不用担心，最多从明天开始我就不去学馆，留在家中也好帮娘亲做些事情，他陈天贵再嚣张，也不敢在光天化日之下前来咱们家闹事吧"
　　楚秀点了点头道："也只有如此了"
　　赵硕不禁道："可是小妹还有一年才能够结业的"
　　赵鸾笑了笑道："二哥，你别忘了，小妹我可是比你聪明哦，该学的东西我都学到了，甚至比二哥你现在学到的东西还多呢，再说没有条件去学馆读书的人多了去了，比起山中乡下那些人来说，我们已经比他们强了许多了"
　　赵硕闻言脸上不禁露出一丝赧然，被自己妹妹如此说，就算是赵硕脸皮厚也会感到不好意思，可是谁让自己的前身的确不怎么聪明呢。
　　见到赵硕那副尴尬的模样，楚秀伸手在赵鸾的额头点了一下道："你这丫头，怎么能那么说你哥哥呢，枉你哥哥那么护着你"
　　赵鸾冲着赵硕吐了吐舌头道："二哥，我可没笑你只是……"
　　楚秀看了赵硕一眼，嘴角露出一丝笑意，拉着赵鸾的手道："好了，你就别解释了，你哥哥受了伤，让他好好的休息吧"
　　接下来几天中，赵鸾便留在家中帮着楚秀做些家务，没事的时候就陪着赵硕，而赵硕也没有浪费如此的机会，从赵鸾的口中得到了许多关于这个世界的事情。
　　传说荒古世界乃是一位上古大神自混沌之中开辟出来的，后来诞生无数的大神通者，这些大神通者参悟上古星空的运转之道继而感悟天地万物大道，当他们感悟出上古星空的奥秘之后，一个个纷纷仿效上古大神从荒古世界的边缘开辟混沌进一步的开拓荒古世界，一边壮大荒古世界，一边试图通过开天辟地来感悟更多的天地至理，完善自己的修行之道来。
　　而这些感悟出数道大道法则并将其彻底掌握，近乎不死不灭存在的大神通者就被称之为道主，意为大道法则之主。
　　当然修行无止境，法则在下，大道至上，当一些道主通过开天辟地走出自己的修行之路的时候，他们所开辟出来的世界便与荒古世界相融，同样诞生出蕴含着他们修行之道奥秘的一片星空来，而能够走出自己的修行之道，开辟出一片不亚于上古神人所开辟的古星空的星空的道主便自然超脱于古星空与一道或数道大道法则相合，成为不死不灭的伟大存在，这样的道主少之又少，因此被称之为大道主。
　　大道主因为触摸到了大道法则之存在，大道之下，法则长存，即为不朽。
　　据说除了开辟古星空的那位大道主之外，另有八位大道主走出了自己的修行之道，当他们完全开辟出新的一方星空的时候，天地震动，一方道韵显化的图卷自星空中诞生，于是被后世称之为传承星图的道韵至宝便出现了。
　　最初的传承星图只有九份，各自蕴含部分大道至理，更有诸多法则加持，正对应一片完整星空星辰运转之秘，可牵引一方星空之力，其强大之处就连大道主都为之侧目，此即为九份荒古级别的传承星图，也是道韵至宝的存在。
　　奈何上古时期诞生了无数的道主强者，这些大神通者法力无边，神通广大，*纵大道法则，运用自如，比之大道主只差一线，却不得长存，同样想要迈出最后一步领悟大道至理，成为那不死不朽的无上存在。
　　然而靠着他们本身的修行不知道要到什么时候才能一窥那无上大道，与大道法则相合，于是九份道韵显化并蕴含诸多法则的传承星图便成为了无上的至宝，或许通过借鉴这些天地至宝，感悟其中大道至理，他们就有机会迈出那至关重要的最后一步。
　　争斗厮杀开始了，能有资格争夺九大传承星图的人哪一个不是万古英豪，千秋俊杰，一个个神通广大无边，足可以毁天灭地，一场绵延亿万年的毁灭性大战，愣是将九大星空无数的星辰打的破碎，这是一个道主遍地走，道尊不如狗的荒古年代。
　　道，是最为玄妙的存在，天地不存而道存，自天地诞生的那一刻起，大道便贯穿古今，无所不在，其玄玄之妙，就连无上存在的大道主都说不清、道不明。
　　在大道伟力作用下，破碎的九大星空彻底的融合在一起，虽然还各自保留着各自的一部分星空特性，但是却的的确确的是彻底的融合，其神秘玄奥，就连那八位大道主也不禁为之惊叹。
　　九大传承星图即便是贵为道韵至宝，可也难免在那异宝无数的上古争斗中破碎成成千上万份散落四方，上古道主更是陨落无数，而那些陨落的上古道主身死之后，其遗骸其大无边，足以抵得上一方星系大小。
　　八位大道主眼看新的星空形成，昔日故友、门人弟子陨落无数，就连当初孕育他们的本源之地也破碎无数，于是八位大道主首次联手将那些陨落在无尽星空之间大多数的道主遗骸施以大神通聚集于本源之地。
　　道主遗骸，精、气、神化为大地、湖泊、海洋、山川草木，无数道主遗骸被强行汇聚，于是本源之地便成为了这片世界最为奇异的存在，这方由无数上古道主遗骸所化的大陆汇聚了这片世界数成的气运，加上每每有道主级别的强者陨落，总会有大神通者将其并入大陆，久而久之，这方大陆其大无边，昌盛无比。
　　当赵硕从赵鸾那里了解到这些的时候，赵硕心中不禁卷起滔天巨浪，遥想上古，那些大神通者颠倒乾坤、一念之间星辰破灭，真可谓是强大。
　　深吸一口气，赵硕看着一脸向往的赵鸾道："小妹，不知道那位最早开辟这方世界的上古大神是何人，说来他才算是这方世界的始祖呢"
　　赵鸾撇了撇嘴道："那位大神开辟了这方世界后就从未出现过，只不过后来人均尊之为盘古父神"
　　*************
　　没收藏的一定要收藏，先养着也好啊，有票砸票`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)

			// 查找第三章和第四章
			var chapter3Index, chapter4Index int = -1, -1
			for i, seg := range result {
				if strings.Contains(seg.Title, "第三章") || strings.Contains(seg.Text, "第三章 怒打纨绔") {
					chapter3Index = i
				}
				if strings.Contains(seg.Title, "第四章") || strings.Contains(seg.Text, "第四章 父神盘古") {
					chapter4Index = i
				}
			}

			So(chapter3Index, ShouldBeGreaterThanOrEqualTo, 0)
			So(chapter4Index, ShouldBeGreaterThanOrEqualTo, 0)
			So(chapter3Index, ShouldNotEqual, chapter4Index)

			// 验证第三章内容
			if chapter3Index >= 0 {
				So(result[chapter3Index].Text, ShouldContainSubstring, "第三章 怒打纨绔")
				So(result[chapter3Index].Text, ShouldContainSubstring, "就在赵硕被众人的目光盯的坐立不安的时候")
				So(result[chapter3Index].Text, ShouldContainSubstring, "收藏，推荐，有票票要砸")
				// 第三章不应该包含第四章的内容
				So(result[chapter3Index].Text, ShouldNotContainSubstring, "第四章 父神盘古")
				So(result[chapter3Index].Text, ShouldNotContainSubstring, "赵鸾将事情的经过讲了一遍")
			}

			// 验证第四章内容
			if chapter4Index >= 0 {
				So(result[chapter4Index].Text, ShouldContainSubstring, "第四章 父神盘古")
				So(result[chapter4Index].Text, ShouldContainSubstring, "赵鸾将事情的经过讲了一遍")
				So(result[chapter4Index].Text, ShouldContainSubstring, "盘古父神")
				// 第四章不应该包含第三章的内容
				So(result[chapter4Index].Text, ShouldNotContainSubstring, "第三章 怒打纨绔")
				So(result[chapter4Index].Text, ShouldNotContainSubstring, "就在赵硕被众人的目光盯的坐立不安的时候")
			}
		})

		Convey("按章节标题切分（英文格式）", func() {
			content := `chapter 1 Beginning
This is chapter 1 content.

chapter 2 Development
This is chapter 2 content.`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)
			So(result[0].Title, ShouldContainSubstring, "chapter 1")
			So(result[1].Title, ShouldContainSubstring, "chapter 2")
		})

		Convey("无章节标题时按长度切分", func() {
			// 构造一个较长的文本，没有章节标题（确保足够长以切分成5段）
			content := strings.Repeat("这是一段很长的小说内容，包含了很多文字和情节描述。", 200)

			result := splitter.Split(content, 5)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 5)
			for _, seg := range result {
				So(seg.Text, ShouldNotBeEmpty)
				So(seg.Title, ShouldNotBeEmpty)
			}
		})

		Convey("目标章节数为 0 时使用默认值", func() {
			// 构造足够长的内容以支持切分成50段
			content := strings.Repeat("这是一段很长的小说内容，包含了很多文字和情节描述。", 1000)

			result := splitter.Split(content, 0)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
			// 如果内容足够长，应该接近默认值50
			if len([]rune(content)) > 50000 {
				So(len(result), ShouldBeLessThanOrEqualTo, 50)
			}
		})

		Convey("目标章节数为负数时使用默认值", func() {
			// 构造足够长的内容以支持切分成50段
			content := strings.Repeat("这是一段很长的小说内容，包含了很多文字和情节描述。", 1000)

			result := splitter.Split(content, -1)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
			// 如果内容足够长，应该接近默认值50
			if len([]rune(content)) > 50000 {
				So(len(result), ShouldBeLessThanOrEqualTo, 50)
			}
		})

		Convey("章节数过多时只保留前N章", func() {
			content := `第一章
内容1

第二章
内容2

第三章
内容3

第四章
内容4

第五章
内容5`

			result := splitter.Split(content, 2)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldEqual, 2)
			// 验证保留的是前2章
			So(result[0].Text, ShouldContainSubstring, "第一章")
			So(result[1].Text, ShouldContainSubstring, "第二章")
		})

		Convey("每个章节应包含标题和正文", func() {
			content := `第一章 开始
这是第一章的内容。

第二章 发展
这是第二章的内容。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			for _, seg := range result {
				So(seg.Title, ShouldNotBeEmpty)
				So(seg.Text, ShouldNotBeEmpty)
				So(seg.Text, ShouldContainSubstring, seg.Title)
			}
		})

		Convey("处理多个连续空行", func() {
			content := `第一章 开始


这是第一章的内容。



第二章 发展


这是第二章的内容。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)
			// 验证空行被规范化
			for _, seg := range result {
				So(seg.Text, ShouldNotContainSubstring, "\n\n\n")
			}
		})
	})
}
