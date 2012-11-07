/* collate.go - collect commit hashes by tag
 *
 * I've annotated a subset of the git log for aoe driver
 * development to categorize commits according to the logical
 * change they affect.  The tag is the name of the category.
 * 
 * In awk, it's easy to do whitespace splits, but not to
 * build a list of commits based on the third ws-delimited
 * field, which is the tag.
 *
 * This program builds and prints a list of commits for
 * each tag, printing a command that will export the collated
 * commits from a git repo.
 *
 * Here's an example test-and-run invocation pair:
 *
 *   go run collate.go < commits.txt | less
 *   go run collate.go < commits.txt | sh
 *
 * The example output is too ugly as is, so here's a cleaned-up
 * version, with abbreviated sha1 sums.
 *
 *   sh export-oldgit.sh 20-dyntgts 1e27fb605589 4cf18aacd132f 2e98529403a 86e78286
 *   sh export-oldgit.sh 21-bkl 362b379651e
 *   sh export-oldgit.sh 22-noto b68f818b81eb
 *   sh export-oldgit.sh 23-comma 27dd17076bc5
 *
 * Here's some example input:
 * 
 * commit 1e204856303d6cdf8eff84bc86428658a8e65549		blkrunq
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Fri May 2 14:04:09 2008 -0400
 * 
 *     use blk_start_queueing when available.
 * 
 * 61
 * 
 * commit def60ed79defcf6e1d8371476def3f90d55b1143     compat
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Mon Jun 2 12:18:29 2008 -0400
 * 
 *     split for_each_netdev from skb_reset_mac_header et al.
 * 
 * 
 * commit 42543ee8d44d8bb8cf575022963a4705efd62aa0     ata_ident
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Mon Jun 2 16:22:55 2008 -0400
 * 
 *     merge sah's ATA device identify exporting.
 * 
 * 
 * 62
 * 
 * commit ab5e4e5384d36962e2b45007aabecb4ecf2d28b4     congestion
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Tue Jun 24 13:54:04 2008 -0400
 * 
 *     congestion avoidance and control.  (conf needs updating.)
 * 
 * 
 * commit 41358436bc701084558c2b55382308ec44ea276d     cleanup
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Tue Jun 24 14:02:18 2008 -0400
 * 
 *     remove unused variables.
 * 
 * 
 * commit 5f5401407168b56b0ebd9f1120a5694a795f79ff     congestion
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Tue Jun 24 14:05:34 2008 -0400
 * 
 *     don't let t->maxout exceed t->nframes.
 * 
 * 
 * commit 5d1cf5a51471c48168051a14a8ffc1efce3e6a42     congestion
 * Author: Ed L. Cashin <ecashin@coraid.com>
 * Date:   Tue Jun 24 14:10:46 2008 -0400
 * 
 *     start with cwnd of 1 and let slow start work.
 */

package main

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"container/list"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	tags := make(map[string]*list.List)
	taglist := list.New()
	line, err := in.ReadSlice('\n')
	for err == nil {
		fields := strings.Fields(string(line))
		if len(fields) > 2 && fields[0] == "commit" {
			hash, tag := fields[1], fields[2]
			if tags[tag] == nil {
				tags[tag] = list.New()
				taglist.PushBack(tag)
			}
			tags[tag].PushBack(hash)
		}
		line, err = in.ReadSlice('\n')
	}

	i := 1
	for e := taglist.Front(); e != nil; e = e.Next() {
		k := e.Value.(string)
		v := tags[k]
		fmt.Printf("sh export-oldgit.sh %02d-%s ",
			i, k)
		for e := v.Front(); e != nil; e = e.Next() {
			fmt.Printf("%s ", e.Value.(string))
		}
		fmt.Println()
		i += 1
	}
}
