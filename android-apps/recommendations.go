//  android-apps/recommendations.go - Sam Rowe's Android app list
//
//  When I tweeted about my new Android phone, my friend Sam Rowe sent
//  me a list of recommended apps.  His email said, "Here's the fun
//  version.  Let me know if you need the unfun version."
//
//  An earlier version of this program printed a histogram of the bytes
//  formed by interpreting each pair of characters in the string below as
//  a hex byte.  The distribution was pretty even, suggesting compression
//  or encryption was involved, assuming it really was his list of app
//  recommendations.
//
//  Fortunately, it was compression, not encryption.  The program below
//  uses the hex package to turn the ASCII pairs into bytes, and then
//  io.Copy sends to standard output the data from a Reader from the gzip
//  package.

package main
import (
	"bytes"
	"encoding/hex"
	"compress/gzip"
	"io"
	"os"
)

func main() {
	s := "1f8b08000000000002039d56c16edc3610bdf32bc6b9780db85ca0470345e1d8496a206d0dd4408ec1481c4a8c288e408e56597f7d87dab55b04b9581771a525dff0cdcc7be4534fc0291e218603c182314e385186874b7da583fe2a32b7032c41fa1bd38b4ce566bf9f221e6dc7dc45b22d8ffb229c698fd354f68e04432cbf07f79bfe639bcc0325219c586ce444f12899c83a1ad9ec1ea0478d23ba89098fa0e14ae07465cc7b2a020d8a503e6a6cd791bc357a22b1fddc60a4c162729983b3758ecdd4704e76ca7c8ef3a4e434d0167adf0ed6636cf9ff705ef9c140c78631bb2da0cfd18634cd3292f4ec6c4409e98c7d088e182ad4b6fd8edf85dadeae3027148bce98272d40c439b57d2d3dcc856017e4b2d48a1c4f0f8f45ae3644142a115bd62f8b7d09712693c9b9500774faededd08e6892cc73a3338f88f98467ccc3e5088905426ae3ec42eaa0c3910a34549477d1760b459b9e008b92c552e691e0c8f365a6f33ae1ba44576ed81526b10b67e74372b68d41bbdf987b6a758456bb3139cce796de809ed9c75ea986a23d875d4533e68e93e621560e19a67ea5961c48c654bc56d4875869f3de6785b016c659d35f88366ca028b2c590574119f3cfdd239c49be0d2b1e0b15f652b5e965fa2989060b39ad14042910b9552170dab0e7811a6cece788e38fc9d272071f4ec0f0f9c33decd6d4acaee439c34817da22bca4d5a31e33af1eb54507598ba53e51e6a86aee5e1d2986ae171f798941c898bf3ede8160195ed5b8c5939635afc9b7d8565effa9eed3ba1cee7a6d03fa2afcf5b1a65965de840ed4a995e3265799e6d237738c242fbcaab5170931c294a9b6e029a6f620ac31578bafa6ab79c553721d79d4e4d4d2aff25338b51d0d07bbea3d94b764bdea4d8de731b41f39569731794e490bb0ffc65d57473df2866a1135d0ea7c5b6a3b272d9a84f6b5ac7a1afc6acc973084895c40d8a9d84ec71c2f4a9a3ddc9e666af2dfa9dfa87eca3bb8b8782b47ce9d5d5ea218739bd61a420a2dfd6836371ba09f99f532609f977aa6fdcc5863484375967a4bf05e5f07820f2a119d45bba217877205b7233eabbafee4466de8dafcf1feef5f3e31dc679e1afe7e0d9dde2ce6e61ad46ee55a259f06206961e943db438dd8cd544a0d562d5a3baa1aaba9ea2d143d045fbfaf47969e96fa2c940f4abf5863fe05336d69b3dd080000"

	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(b)
	g, err := gzip.NewReader(buf)
	if err != nil {
		panic(err)
	}
	defer g.Close()
	_, err = io.Copy(os.Stdout, g)
	if err != nil {
		panic(err)
	}
}
