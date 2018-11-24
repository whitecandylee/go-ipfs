package commands

import (
	"fmt"
	"io"

	core "github.com/ipfs/go-ipfs/core"
	cmdenv "github.com/ipfs/go-ipfs/core/commands/cmdenv"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	tar "github.com/ipfs/go-ipfs/tar"

	"gx/ipfs/QmQtg7N4XjAk2ZYpBjjv8B6gQprsRekabHBCnF6i46JYKJ/go-path"
	cmds "gx/ipfs/QmVKobNrLGkvfUPtnku5fdm4nCrd7fpTFCsA69JazgTCQa/go-ipfs-cmds"
	dag "gx/ipfs/QmdURv6Sbob8TVW2tFFve9vcEWrSUgwPqeqnXyvYhLrkyd/go-merkledag"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

var TarCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Utility functions for tar files in ipfs.",
	},

	Subcommands: map[string]*cmds.Command{
		"add": tarAddCmd,
		"cat": tarCatCmd,
	},
}

var tarAddCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Import a tar file into ipfs.",
		ShortDescription: `
'ipfs tar add' will parse a tar file and create a merkledag structure to
represent it.
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.FileArg("file", true, false, "Tar file to add.").EnableStdin(),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		it := req.Files.Entries()
		if !it.Next() && it.Err() != nil {
			return it.Err()
		}
		if it.File() == nil {
			return fmt.Errorf("expected a regular file")
		}

		node, err := tar.ImportTar(req.Context, it.File(), nd.DAG)
		if err != nil {
			return err
		}

		c := node.Cid()

		return cmds.EmitOnce(res, &coreiface.AddEvent{
			Name: it.Name(),
			Hash: c.String(),
		})
	},
	Type: coreiface.AddEvent{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, out *coreiface.AddEvent) error {
			fmt.Fprintln(w, out.Hash)
			return nil
		}),
	},
}

var tarCatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Export a tar file from IPFS.",
		ShortDescription: `
'ipfs tar cat' will export a tar file from a previously imported one in IPFS.
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("path", true, false, "ipfs path of archive to export.").EnableStdin(),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		p, err := path.ParsePath(req.Arguments[0])
		if err != nil {
			return err
		}

		root, err := core.Resolve(req.Context, nd.Namesys, nd.Resolver, p)
		if err != nil {
			return err
		}

		rootpb, ok := root.(*dag.ProtoNode)
		if !ok {
			return dag.ErrNotProtobuf
		}

		r, err := tar.ExportTar(req.Context, rootpb, nd.DAG)
		if err != nil {
			return err
		}

		return res.Emit(r)
	},
}
