import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// One glyph per terminal column. Colors are stored left to right for visible glyphs.
const artwork = [
  ["                                                        .--.", "2b4e80286ca2286ea42c548a"],
  ["                                                        .--..         .==-", "2b5284296fa62871a92e5a90283c6b253f682bbced2bc2f3217094"],
  ["                                                        .....        -=++:.", "2a3b6c284a812e4f862c4677283b6725406a2bb6e52edeff2cdfff2878a4264972"],
  ["                                                      .:--:..        .-=-..", "39457e485f953d628e3966933e77b32e56962b3b78254d7622799e2ab8e821739926547f253c67"],
  ["                                                   .-*#%%%##+-..      ....", "3b4880586e9695d2f99affff8bffff76ffff6dffff63ffff53ffff42d3f6276ea6283c7e32396e254671264d77264872253e67"],
  ["                                                 :+%%%########+:..                -=*-.", "4a5a908bb3dbacffff82ffff62ffff59ffff5affff5cffff5cffff59ffff59ffff59ffff57ffff42dfff275ca43138762f396722567a1abeda14ffff18829a253f64"],
  ["                                               :*%%%@@%########*:..             -=*#*-.", "4e5f94acf6f6b2ffff7cffff88ffffbcffffdbffff9cffff5effff62ffff60ffff5dffff5cffff58ffff56ffff57ffff4dffff2877c2313878323b6c2448701ac4df13ffff13ffff14ffff197892244e74"],
  ["                                             .+%%%@@@@@##########-..            .==-:.", "4250899ab7dbbdffff9cf6f6a2ffffffffffffffffffffffffffffbbffff5affff62ffff60ffff5dffff5cffff58ffff56ffff54ffff56ffff50ffff246cab30387b343c6d24547a207e9e1bbeda187d9721678a244e74"],
  ["                                          ..=%%%@@@@@@############-..            ...", "2f386d2f3c7a8299c5ccffff89ffff76ffffffffffffffffffffffffffffffffffc5ffff61ffff62ffff60ffff5effff5dffff5cffff59ffff57ffff56ffff56ffff56ffff53ffff2773b32e387c353c7024486c244e7224496e"],
  ["                                        . :#@#%@@@@@%##############-..", "2f3867516299cbf3ffbdffff5cffff88ffffffffffffffffffffffffffffffffff8effff59ffff61ffff61ffff60ffff5effff5dffff5cffff5cffff59ffff58ffff57ffff57ffff57ffff56ffff2b81c22a3a81363c71"],
  ["                                       ..+@%##@@@@%#################=..", "2f396d2b397a8fa1c7e6ffff85ffff4dffff50fffff4fffffffffffffffff2ffff99ffff5dffff59ffff60ffff60ffff60ffff5effff5effff5dffff5cffff5affff59ffff58ffff57ffff56ffff56ffff54ffff54ffff3abbff274694343a74"],
  ["                                      ..#@#####%##################**#+:..               .==+*+.", "2b38723f508ed2ecffd4ffff56ffff4cffff67ffff60ffff6bffff7effff6bffff56ffff57ffff5effff5effff5effff5effff5effff5effff5dffff5dffff5cffff5affff59ffff59ffff58ffff57ffff54ffff53feff53fdff56ffff46dbff275eab303877313a6c24466c1abbcf1abbcf15deee14ffff17dbed244066"],
  ["                                    . -@%*%@@@####################*****-..             .-****=.", "2f3968a4a6a6fcffffaeffff48ffff8bffffffffffffffffbaffff52ffff59ffff5affff5dffff5dffff5dffff5dffff5cffff5dffff5dffff5dffff5dffff5cffff5affff5affff59ffff59ffff59ffff58ffff58ffff57fdff54faff53f8ff54fcff50f4ff297ac1273e8a323970244f740b999912ffff12ffff12ffff13ffff18d6e8225274"],
  ["                                   . =@%*%@@@####################*******+:..           ..--:..", "303a6c8a99c6ffffff90ffff44faff8fffffffffffffffffffffff71ffff59ffff5dffff5dffff5dffff5effff5dffff5dffff5dffff5dffff5dffff5cffff5cffff5affff5affff59ffff59ffff59ffff58ffff58fdff57fcff57faff56f8ff53f6ff52f4ff53f4ff54f9ff44ccff2c67ba2c387731396c254368225d7f19738a18798e206585245477254368"],
  ["                                  . +@#**#@%##################************=...", "323c6e9cacd4ffffff7bffff4ff8ff54fcff88ffffd6ffffa0ffff5effff5affff5effff5effff60ffff60ffff60ffff60ffff60ffff5effff5effff5dffff5dffff5cffff5cffff5affff5affff59feff58fcff58faff58f9ff57f8ff58f7ff56f4ff54f3ff54f0ff53efff53eeff54f2ff52eeff39a4ff284c9a3036722f3868"],
  ["                                 ..*@#******#################**************+:..", "313b6d2e323cb0bde5ffffff72ffff50f4ff5af8ff5afaff58f9ff53f8ff57faff5efeff60feff60fcff60fdff60fdff60feff61ffff61ffff61ffff61ffff61ffff60feff5efeff5efeff5dfdff5dfcff5cfcff5afaff5af9ff59f7ff58f8ff58f7ff54eeff57f3ff58f8ff57f2ff56edff54eaff54e8ff54e8ff56eeff49cff62e74cd283980313a6e"],
  ["                                ..+@#*****#****#*########********:..:+*******=...", "313a6b3539489eaad6ffffff77fdff50efff5af6ff5df7ff5ef8ff5ef8ff5efeff5df9ff59eaff59e9ff5ef7ff60f9ff60f8ff61f9ff61faff62faff62faff62faff62faff62f9ff61f9ff60f8ff5ef7ff5df7ff5df6ff5df6ff5cf4ff5af7ff52dfff2a6db520569d21579d2c7cca4cd0ff59eeff57e9ff57e8ff57e6ff56e3ff57e5ff5aeaff42b2ff264ea22f36752f3867"],
  ["                                .-@%******=..:+#*****************++=:-*********:..", "2b387aa6a6a6ffffff99ffff4ee9ff5aefff5ef3ff60f4ff60f4ff5ef8ff44b2ff255c9f22579d2660a549b7f662f8ff61f3ff61f4ff63f4ff63f6ff63f6ff63f4ff62f4ff62f4ff62f3ff61f3ff61f2ff60f2ff60f2ff5ef0ff5df0ff5cefff57e9ff4fd4ff4cd0ff3fb6ff2b74c42f78ba59edff58e8ff59e6ff58e6ff58e4ff57deff57dcff60e5ff5ad6ff2e6ccc2a367c323c6e"],
  ["                               ..@@*******=+*********************++*************- .       .-++++-", "353f7736458cffffffdaffff4ee2ff5ae9ff60eeff61efff61efff61efff5eeeff42a3fa4abaff57e2ff56e5ff5eedff63f2ff63eeff63eeff66efff66efff66efff66efff66efff66efff66eeff64eeff63edff63ecff62ecff60ecff60efff59edff4edcff46d2ff4ad6ff53e5ff59f3ff5cedff59e5ff59e5ff5ae5ff5ae4ff5ae4ff59e0ff56d6ff53d5ff60dbff70e5ff346fc3323c7124507617819917e3fc17e5fd17e4fc16d8ef168ba3"],
  ["                              . *@**********+*=+***************+*#+--+********+**- .     ..=****+", "313a6bc0c8eeffffff6ee9ff59e0ff5ee5ff62e8ff64e9ff66eaff66eaff66eeff53daff50b6f976c0ff7bbcf961adf752b8ff64e4ff6becff68e8ff68e8ff6ae9ff6ae9ff6ae9ff68e8ff68e8ff67e8ff66e8ff64e6ff62e5ff61e9ff49bef563aaecadd1fac8dffe97b0d64a6b9b2b68a64dc5ff5ee8ff5ee2ff5de2ff5de0ff5de0ff5ce0ff5ad8ff59d1ff58d0ff63d4ff7fe5ff3d72c6343e712542682261841ac4df13ffff13ffff13ffff13ffff16d1ea"],
  ["                              ..@@*******+*@%#+..+***********+*@@*@#. =*******+**#-..     ......", "343f773a4c91ffffffe3ffff59d6ff66dbff66dfff67e0ff68e0ff6be2ff6be8ff51b3f6aac5ecffffffffffffdfdff2acb2ce31427225569864cfff71e5ff70e2ff70e2ff6ee0ff6de0ff6ee0ff6ee0ff6de2ff6be0ff6ae0ff68e3ff4fb2f6bed1f3ffffffffffffbac2dcffffffdce3f9273a6e4aaeff64dfff64dbff64daff63daff63daff62d8ff5eccff5dcaff5dc8ff71d1ff92e3ff375cac2c3a7e303968253f67244c7224527724537824527624486e"],
  ["                              -=@#******+*@%:@@: :************@@.-@%. .******++++**..", "2b387a7e8bc1ffffffa3efff68d0ff6ed5ff71d8ff72d8ff72d8ff71d9ff5fc0f6b6ccefffffffffffff455688ffffffffffff586c99346ab378deff76d9ff76d8ff75d8ff75d8ff75d8ff75d8ff74d9ff74d8ff71d8ff67d1ffa0c7fcffffffffffff3a52844f5d81ffffffeef4ff364e8225539a68d2ff6bd2ff6cd1ff6cd1ff6cd0ff6bcfff63c5ff62c1ff66c1ff60baf685d1ff93ceff284399354076"],
  ["                             ..+@*******+@@+ ..: .***********%@@    - .******+++++#-..", "2f3866283980a4addcffffff92d6ff77caff7cceff7cceff7eceff7eceff7fcfff6db7f6ffffffffffff99a6c630487a31487b576b982a539c82d4ff82d4ff81d2ff80d1ff7ed0ff7ecfff7ed0ff7fd1ff7fd4ff80d4ff72c8ffdafcffffffffffffff5462832a56a276c8ff76c8ff76c8ff76c7ff76c6ff75c5ff6cbbff6ab7ff6bb7ff64b0f671b4f6acdcff4864b82b3a7c2f3866"],
  ["                               +@********#@@.   .=+-=++++=--+*@@+    -=****+++++++*+ .", "a0a7d7ffffff95caff82c2ff86c2ff86c2ff88c2ff89c4ff89c2ff81beffc2ecffffffffffffff3446750f1725689cff76aef64970c7649cff74b0ff7ab6ff74b0f66eaaf66ba4ff4570cb426abc74b2f694d1ffffffffffffffa4b1cc1d3f85669eff7fbeff80bdff80bdff80bdff7bb5f677b2f672adff71acff72acff6ea6f670a6f6add0ff88a3ff323b6c"],
  ["                              .-@*+*******#%%=:-+*-         =**#@##:-+**++++++++++*+..  ....", "2c397b5f689cffffffaacaff8cb6ff8fb7ff92b7ff93b8ff8fb7ff98baffa4bdffa3bbffbdd2ffecfeffe4f4ff889ce24f68b84661b680a8ff98c5ff415db46c98ff8bbeff8fbdffc5e4ffffffffc6d8ff54669d4366c5506cc693b0ffa2bcff99b8ff8ab3ff85b1ff88b1ff81aaf678a1f678a2ff7aa2ff7aa0ff769af6769af6b1c6ff95a6ff273b86364071244970224f75224870253f66"],
  ["                              ..@#+**+#%%@%%******+  ==+**-.+++++**#@@%%%*++++++++#= ...-**+=-", "343f7732448fffffffd5d9ff95a7ff9db0ff9eaeff99acfff6e0f6ffe8ffffe8ffffe9ffffe3ffffdbffc4c1ffa2b7ffa3b6ffa0b4ff9cb1ff9db3ff8398f6c17eb1c17eb1ea8bbbff95ccff99ce8858802b44938ba7f68fabf694aeff8faaff8fa9ff98b2ff9ab6fff4d8ffffecffffeaffffe5ffffe8ffffdeffbbb6ff8aa6ff8399f67c93f68198ff8198ff8196ff7c8ff67d90f6c2c7ff666ebd313b6c263f67225a800b999912ffff12ffff13f0f617d1ee177d97"],
  ["                                =@**++#%%%%%*++++++=:*#%#*-=+++++++#%%%%%*++-====+#:.  .==+*#*.", "908fd8ffffffada6ffa59ffba29af69f98f5f6d6f6ffdeffffe3ffffe3ffffe0ffffd6ffc6aeff9e9cf6a19df6a4a3ffa4a3ffa4a4ffa1a0f6777eff4f519dffa0ceffc0e8ffc2eaffbde4ff9dcb685599848fff99a0f6989ef69ea4ff9da4ff9da3ff959af6959cf6edc5ffffdbffffe3ffffe6ffffdeffffd2ffbcacff8690f9888eff898fff858aff8288ff868aff8689ff9798f6cec8ff505cc22f3c78244c7118c2df18c2df15e0f613ffff12ffff13feff24486d"],
  ["                               ..*#++++++++++++++++++=====+++++++++++++++====+%##*+..     ...", "323c7128398bcebaffe9c4ffaa8af6ad8df6b696ffb194ffb395ffb694f6ba95f6bb99ffb094ffb095ffac91f6ab91f6b196ffb098ffae98ffa893f6ac97f6a290ff8478ff8a77ee9278e88575f07c78ff9c92ffa899ff9e91f69e91f6a498ffa498ffa496ff9d91f69c90f69c92ffa394ffac98ffa791f6a08ff6958bff8c85ff8a81ff8b82ff867cff9f94fffcf7ffd9d1ff635abdbdb2ff9e9dff253889343f7125406724486c224e74"],
  ["                                ..**+++++++++++++++++++++++++++++++++++=====+@@++*..", "303e74786299c8a4ffe0a4ffb177f6b87fffbd86ffbe8affb684f6b584f6bd89ffbe8affbe8affb686f6b586f6bb8bffba8bffb88cffb287f6b189f6b78fffba90ffb68fffac89f6ac8af6b38fffb08effad8bffa786f6a686f6ac8bffac8bffac8bffa485f6a385f6a989ffa889ffa485ff987df6967fff9078ff8f77ff8c76ff886dffa694f6ffffffffffffb29fffb2a0ffbab0ff35429f303e76"],
  ["                                 ..=*+=+++++++++++++++++++++++++++++=======#@@**+...", "2f3c722e398b9e78ffdf93ffc778ffbd71ffc076ffbd77f6bf7af6c67effc67fffc680ffbf7cf6be7cf6c581ffc282ffc284ffba7ff6b97ff6be82ffbd82ffbc82ffb47df6b27cf6b781ffb681ffb481ffad7cf6ab7cf6b180ffb181ffb180ffaa7cf6a77af6a87cffa077ff9872ff9370ff926eff906eff8a67ff9575ffd8c4ffffffffffffffbca7ffb6a2ffa29cf63a4fa72b427b2c3966"],
  ["                                  ..:=++==++++++++++++++++++++===========*@@#++=%:#..", "2e396c27397f614dc6b76bffcd70f5c46af5c46bffc26bffc56effc771ffc875ffc373f6c474f6ca78ffc87affc87affc077f6bf77f6c47bffc27affc17affb774f6b673f6bc78ffbc78ffba78ffb173f6b073f6b376ffae75ffa974ffa36eff9c6cff966aff9468ff936aff9066ff8c62ff946cffd1b7ffffffffffffffdfccffa68cf5a997f57786f92b65ac266b9a2480a8225980253f67"],
  ["                                    ..==++=============================+=#*++=:.:*++=.", "2b3a6c323c8fad62ffad62ffd06dffd470ffcb6affc467ffc068ffc068ffc06affc16cffc26dffc16effc16effc06effc070ffbe70ffbd70ffbb70ffb870ffb46effb16dffad6cffa96bffa668ffa067ff9d66ff9a64ff9866ff9666ff9466ff9366ff8f63ffa27affb191f69871ffd9c1ffcbaeffb294ffa994ff847ffe4352ae2a528927689117daf914fdff14fcff1ab7d822577c"],
  ["                                     ....:==++=========================++==:......:++=", "2e39672c3c6e2e3b7c4942ac744edba860ffc26affcc6bffce6bffca6affc466ffbd64ffba63ffb463ffb263ffb063ffad62ffac63ffaa63ffa863ffa663ffa363ffa063ff9f63ff9d62ff9a62ff9861ff9661ff9461ff9362ff9363ff9466ff986bffa47cffae8bffa688ff9680ff8177ff595dc435408c2c3a752f3a6c2c396827426b24527a1e83a716daf513eff51ac2e0"],
  ["                                      ......::--=====================--::.....     ..", "2f3868323b6e313e722c3c742e3b7f3b3e905045b0674bcc6e45c57e49d1a85effb363ffb866ffba67ffba67ffbb6affba6affb768ffb66affb36affb16bffad6affa96affa76bffa66cffa36dff9e6cff9c6eff9870ff9370ff886eff645cd15452bd5057c44452a93440862f3b7a303b74323c7131396c263e64253f64"],
  ["                                       ..............::::::::::........:......", "30386a31396c323a6d343a71323c74313c742e3c742e3c742f3e7c323e82383e8a3e3f9044429d4944a84f46b15346b45548b65648ba5748bf5749c0564ac4514abc4f4bb74c48b34648ad4246a23c449939438f35438a3045882c4d892b57922b5b93304e88344277343b7131396d31396b30386a"],
  ["                                                .........   ..........%:#.", "2f38672f38672f39682f39682e39682e39682c39682e39682c396a2c39682e396a2e39682e39682f39682f38682f38682f39682e3e6d2b45772573ab2375af2274ad2a4272"],
  ["                                                                      .::.", "25406e1f6da22171a6245d8c"],
  ["", ""],
  ["      ..             ...", "2c3c6d2c3c6d2b3b6b2e42742c3f70"],
  ["  .:-=====:-=-    :-###+==:.    -=--=+*=: -=+*+-      ==+*+-.   .==.-+*+- .-+*+-.     -=+*+=:    .==.-*+: ==.     -=:", "2c3b723e64d03d6cd15395ff549aff549cff529aff4c8cff3d64bc3b6dcc509cff3d6ac33859a8386bc74d9afa50a5fa51a7fa54acff52a6ff4994ff3e6dc42f4275454d70949cb46469823e456a828aa8b0b6d0b4bbd5989fbb5d64853e456b858ba9adb4cfb4bcd699a1bb5a617b9098b49098b4acb3cdbbc2dba7aec86168813c456a363f638b93b18a92b142496e5d627da9b1cbb6bdd69aa1bb595f79323960646982a4abc5b6bdd6a7adc86469823940644a52758088a4a7aec8b7bed8a8aec88c93b0555c7d3f45679096b18086a4343b5d595f78a1a7bfaab1c6575d7e8188a38e94ad393f60555a739399ae4f5572"],
  [" .###:..:-====  .-==-:.:-=+=:   *@@@@@@@@%@@@@@@@:  -@@@%#@@@+ .=@@@@@@@@@@@@@@@@-  :#@@@@@@@%-  +@@@@@@=.@@%    =@@*", "3950a75085fa548ffa5186fa4064cf38509f344c983656a8395db95399ff539cff539aff5394ff2c3b6d3c6dd154a0ff53a0ff3b69cb385bab3652983859a3345daf539dff53a6f652a0ff3b60adc5cbe5fffffffffffffffffffffffffffffffffffffffffffffffff7feffffffffffffffffffffffffffffffffffffffffffff565d805d627dfffffffffffffffffff3f9ffe6edffffffffffffffffffffadb3ce14151a8a92b0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff6166804f5678e4eafefffffffffffffffffffffffffffffffffffffffffffeffff5d647db2bad2ffffffffffffffffffffffffffffffffffff8a90ac494f70fffffffffffffdffffa6a6a6ffffffffffffb8bed6"],
  [" -=-.     .###  -==:     .###.  *@@*  .#@@%.  +@@# :@@@=::-*@@*.=@@%.  +@@@:  -@@@ .@@@-   :%@@- *@@%:    =@@#  :@@#", "4367d15381ff4467d131407a354a8f5189fa528bfa5183fa3d61c35296ff5394ff4063be3140744d8ffa4e99fa5097fa304075c7cfe8ffffffffffffbac0d961646fd1d8efffffffffffffe9efff383f63989eb9ffffffffffffd5dcf45a6182ffffffffffffffffff8088a65e64856067887a7d8ccbd1eaffffffffffffbdc4de14151a8b92b0ffffffffffffeef6ff3a4266979db7ffffffffffffffffff4a5072a6a6a6ffffffffffffffffffa6a6a6ffffffffffffffffff5d647d505676f6fcffffffffffffff5c6179b6bcd5fffffffffffff2f8ff4b5070828aa6ffffffffffffd8def4626985ffffffffffffe4eaff"],
  [".---      .-=-  -==:      -==.  *@@:  .=@@+   :@@# +@@@%%@%%%%+.=@@+   :@@#   .@@@ -@@#     =@@* +@@-      +@@+.@@%", "2e396b4661d14560d1455dd1323f7f4368d15482ff4467d13e5fd05389ff5689ff3c53a43f6bd1508eff5390ff31427bc7cee6ffffffffffff60688615161a8e95b2ffffffffffffa3a9c6505778ffffffffffffe2e8ff979cb7fffffffffffffffffff6fcfffffffffffffffffffff7fdfff9ffffffffffa9b1cc14151a8a92b0ffffffffffffa8aeca4e5576ffffffffffffdee4fca6a6a6ffffffffffffffffffa6a6a6ffffffffffffd8dff68c93b0ffffffffffffb4bbd4b2b8d2ffffffffffff656a819ea4beffffffffffffa6acc4404666fffffffffffffaffff"],
  [" :---....:----  .-=-:....-==:   *@@-  .=@@+   :@@# .@@@=...-*= .=@@+   :@@#   .@@@ .%@@*:.==@@@: *@@-       *@@@@@.", "3e4cb6485ad1485cd14051d13c49b2313a7f2f3878323c893f4fc2455dd14560d14661d1455fd13242844462d1567fff4362d13e54c43240842e3a722f3c7c394da83f5fd15385ff5385ff4261cccad0e9ffffffffffff585d7616171b959db8ffffffffffffaab1cc575e7effffffffffffe4eaff444c6effffffffffffffffff898fad4a52743c4468484e71616781c1c7de969ebb14151a8b93b1ffffffffffffaeb4d0555d7cffffffffffffe3eaffa6a6a6ffffffffffffffffffe8eef6f0f7ffffffffffffffb4bbd4596080444c6d4f57779096b2ffffffffffffffffff4e5472b8bed6ffffffffffff666b82bbc1d9ffffffffffffffffffffffffffffff3b4260"],
  ["  .:------:---    :-------:.    +@@:   -@@=   :@@*  .*@@@@@@@*  -@@=   :@@*   .@@%  .+@@@@@@@*.  +@@-       .#@@@.", "3039804450db444fd14854d14954d14852d14754d14250d14453d14654d14758d14655d13f56d04157d1475fd14860d1465fd14761d14762d1445dd14861e3354590aeb6d0ffffffffffff575e7d686e86ffffffffffff9399b64b5071ffffffffffffc7cfe83f466abdc4dbffffffffffffffffffffffffffffffffffffffffffbcc4dc626981ffffffffffff969eba4a5071ffffffffffffc7cfe6a6a6a6fffffffffffff3f9ff0a0b0c9ca3bbffffffffffffffffffffffffffffffffffffffffffc0c6dc43496a9ea4beffffffffffff585d74646772d9dff6ffffffffffffffffff4a506d"],
  ["   ..     .--:       ....        :.     ::     .:      ::-::     ::     .:     .:.     .:-:.      :.        .%@@:", "363a8e3238764046b14850d14a50d14f58ee313c8634408e343f8a2f3976545c7a464e6e4d53754c5472444a6d565d7c4d53746067855a6079636b894c53744e54764f5573444a6d565c7b3a4263545b78353c5e444c6b606684595d756067844c5270505775494f6ee2e8f6eaf0ffffffffffffff5b607d"],
  ["  :--:::::--:                                                                                              :@@@-", "5656fd4f4ad14b49d14f4eef4746d44747d64d4de65754ff4d4bd14b4ad14c4ee4626984ffffffffffffffffff585d73"],
  ["   ..:::::.                                                                                                :#*:", "36398e4544c24a47de4f4ceb524ef44f4df24a48d83f3fae5f6580cfd5e5bdc4d8555d76"],
  ["", ""],
  ["    ::     :..: ::.:...:. : :..:: :.  -.:.: :.-.:#.::.:. .:.::.:#  .- :  :..::.:...:#..:.  :..:.:: ::.:..:.   .-", "3b62c93d63c75b6487434c6d464f71565e815b64865a63873139595a6388454d70444c6e383f61565e82464f725e678a5d668b484f74383f61576084535c805d668b3a4264545c7a495277566085444d715f698e606a8f464e72515a793a43664f587c4e587c3c45685d668b576084343b5e576186353c613e456a5c648b31395c535d804a537642496d576185626b9140486c545d7d4b537750597e3e456a363f624a537650597f3e466a525a804a52763e456a444c70565e8150597e464e72464f724f587b3f486a5e688c3f466a42496c4d5678434c6e4a53745963855a63874f57793a42624b5274484f71323a595963853a42625e3c9f6e3eb6"],
  ["    ..     . .. .. ... .. . .. .. .   ..... ... .. .. .  .. .....  ....  .. .. .. .......  ....... ..    ..   ..", "2e448830468b363e5d343a5a424a6b3a42623a406242496b3e4466323a5a3c44663f4668383f62434a6d3b4364383f61464f72353c5e40496c444c70383e61363e61444d713b4366434c703f486b3e456a3f466b3b44673e466a3c446831395c323a5d3c45683f486b323a5d3b44673e456a3c45683e4568343b5e3940633f466a363e6140486b3e456840486b3c4466343b5d363c603b4366383f61363e6042496c3942633b4364383f61343b5c40496b40486a383f6040486a383f603f4868444c6c343a593f3070573993"],
] as const;

const widthOfArt = Math.max(...artwork.map(([row]) => row.length));
const raster = artwork.map(([row, colors]) => {
  let offset = 0;
  return Array.from(row, glyph => {
    if (glyph === " ") return undefined;
    const rgb = colors.slice(offset, offset + 6);
    offset += 6;
    return { glyph, rgb };
  });
});

export function renderBanner(width: number, noColor = false): string[] {
  if (width < 1) return [];
  const columns = Math.min(Math.floor(width), widthOfArt);
  const scale = columns / widthOfArt;
  const height = Math.max(1, Math.round(raster.length * scale));
  const margin = " ".repeat(Math.floor((width - columns) / 2));
  return Array.from({ length: height }, (_, y) => {
    const row = raster[Math.min(raster.length - 1, Math.floor((y + 0.5) / scale))];
    let line = margin;
    let previous = "";
    for (let x = 0; x < columns; x++) {
      const cell = row[Math.floor((x + 0.5) / scale)];
      if (!cell) {
        line += " ";
        continue;
      }
      if (!noColor && previous !== cell.rgb) {
        const rgb = [0, 2, 4].map(i => parseInt(cell.rgb.slice(i, i + 2), 16));
        line += "\x1b[38;2;" + rgb.join(";") + "m";
        previous = cell.rgb;
      }
      line += cell.glyph;
    }
    return line.trimEnd() + (previous ? "\x1b[0m" : "");
  });
}

// La versión la exporta el launcher pi-gomemory; "dev" o vacía no se muestran.
export function bannerVersion(raw: string | undefined): string {
  const v = (raw ?? "").trim();
  return v === "" || v === "dev" ? "" : v;
}

// `mem --version` imprime "gomemory 2.54.1"; se queda con el número.
export function gomemoryVersion(raw: string | undefined): string {
  const v = (raw ?? "").trim().replace(/^gomemory\s+/i, "");
  return v === "" || v === "dev" ? "" : v.startsWith("v") ? v : "v" + v;
}

export const HCD_TAGLINE = "HCD · Harness Continual Development";

export function bannerTitle(width: number, noColor: boolean, version: string, gomemory = ""): string {
  const text = "HCD · pi - gomemory" + (version ? "  " + version : "") + (gomemory ? " · gomemory " + gomemory : "");
  const margin = " ".repeat(Math.max(0, Math.floor((width - text.length) / 2)));
  return noColor ? margin + text : margin + "\x1b[1;38;2;34;227;247m" + text + "\x1b[0m";
}

export function bannerTagline(width: number, noColor: boolean): string {
  const margin = " ".repeat(Math.max(0, Math.floor((width - HCD_TAGLINE.length) / 2)));
  return noColor ? margin + HCD_TAGLINE : margin + "\x1b[2;38;2;145;169;197m" + HCD_TAGLINE + "\x1b[0m";
}

export default function (pi: ExtensionAPI) {
  pi.on("session_start", async (_event, ctx) => {
    if (ctx.mode !== "tui") return;
    const noColor = Boolean(process.env.NO_COLOR);
    // Si Pi se lanzó sin el launcher pi-gomemory, las variables no existen: se consultan.
    const ask = async (cmd: string, args: string[]) => {
      try {
        const r = await pi.exec(cmd, args, { timeout: 2000 });
        return r.code === 0 ? r.stdout : "";
      } catch {
        return "";
      }
    };
    const version = bannerVersion(process.env.PI_GOMEMORY_RUNNING_VERSION || (await ask("pi-gomemory-setup", ["version"])));
    const gomemory = gomemoryVersion(process.env.GOMEMORY_INSTALLED_VERSION || (await ask("mem", ["--version"])));
    ctx.ui.setHeader(() => ({
      render: (width: number) => [...renderBanner(width, noColor), bannerTitle(width, noColor, version, gomemory), bannerTagline(width, noColor)],
      invalidate() {},
    }));
  });
}
