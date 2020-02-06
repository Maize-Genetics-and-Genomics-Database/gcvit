/*
	route handlers for the gcvit server
*/

package gcvit

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/awilkey/bio-format-tools-go/gff"
	"github.com/awilkey/bio-format-tools-go/vcf"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"math"
	"strconv"
	"strings"
	"time"
)

// GetExperiments is a GET path that returns a JSON object that represents all the currently loaded datasets GT field headers
func GetExperiments(ctx *fasthttp.RequestCtx) {
	//start time for logging
	start := time.Now()
	// Populate experiments if it hasn't been already
	if len(experiments) == 0 {
		err := PopulateExperiments()
		if err != nil {
			ctx.Logger().Printf("Error: Problem populating experiments: %s", err)
		}
	}

	//Iterate through experiments and build response
	opts := make([]ExpData, len(experiments))
	i := 0
	for key := range experiments {
		exp := ExpData{Value: key, Label: experiments[key].Name}
		opts[i] = exp
		i++
	}

	if authState := ctx.UserValue("auth"); authState != nil {
		// extend slice
		auth := (authState).(string)
		nOpts := make([]ExpData, len(experiments)+len(privateExp[auth]))
		copy(nOpts, opts)
		opts = nOpts
		for key := range privateExp[auth] {
			exp := ExpData{Value: key, Label: privateExp[auth][key].Name}
			opts[i] = exp
			i++
		}
	}

	//Response
	optsJson, err := json.Marshal(opts)
	if err != nil {
		ctx.Logger().Printf("Error: Problem converting experiments to JSON: %s", err)
		ctx.Error("Problem getting experiments.", fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetContentType("application/json; charset=utf8")
	fmt.Fprintf(ctx, "%s", optsJson)
	//Log response time
	ctx.Logger().Printf("%dns", time.Now().Sub(start).Nanoseconds())
}

// GetExperiments is a GET path that returns a JSON representation of all the passed experiment's
func GetExperiment(ctx *fasthttp.RequestCtx) {
	//Start time for logging
	start := time.Now()
	//Parse arguement from path (/api/experiment/:exp
	exp := ctx.UserValue("exp").(string)
	//Populate experiments if it hasn't been populated already
	if len(experiments) == 0 {
		err := PopulateExperiments()
		if err != nil { //log errors opening provided files for debugging later
			ctx.Logger().Printf("Error: Problem populating experiments: %s", err)
		}
	}

	// populate experiment if possible due to auth state
	var expSet DataFiles
	if experiments[exp].Genotypes != nil {
		expSet = experiments[exp]
	} else if authState := ctx.UserValue("auth"); authState != nil && privateExp[(authState).(string)][exp].Genotypes != nil {
		expSet = privateExp[(authState).(string)][exp]
	} else {
		ctx.Error("Experiment Not Available", fasthttp.StatusUnauthorized)
		return
	}
	//Iterate through passed experiment and build response of GT headers
	gt := make([]ExpData, len(expSet.Genotypes))
	for i, v := range expSet.Genotypes {
		gt[i] = ExpData{Value: v, Label: v}
	}

	//Response
	gtJson, err := json.Marshal(gt)
	if err != nil {
		ctx.Logger().Printf("Error: Problem converting genotypes to JSON: %s", err)
		ctx.Error("Problem populating experiment", fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetContentType("application/json; charset=utf8")
	fmt.Fprintf(ctx, "%s", gtJson)
	//Log response time
	ctx.Logger().Printf("%dns", time.Now().Sub(start).Nanoseconds())
}

//GenerateGFF takes a post request for a given vcf file and returns a GFF
func GenerateGFF(ctx *fasthttp.RequestCtx) {
	//Log request received
	ctx.Logger().Printf("Begin request for: %s", ctx.PostArgs())
	start := time.Now()
	//Struct for holding Post Request
	req := &struct {
		Ref     string
		Variant []string
		Bin     int
	}{}

	//parse reference, both ref and variant are in the form "<exp>:<gt>"
	//Peek is used here, as current GCViT only supports a single reference
	req.Ref = string(ctx.PostArgs().Peek("Ref"))

	//parse variant(s)
	vnts := ctx.PostArgs().PeekMulti("Variant")
	for _, v := range vnts {
		req.Variant = append(req.Variant, string(v))
	}

	//parse bin size if available, if not passed, default to config.binSize (500000) bases
	if bSize, _ := strconv.Atoi(string(ctx.PostArgs().Peek("Bin"))); bSize > 0 {
		req.Bin = bSize
	} else {
		req.Bin = binSize
	}

	ref := strings.Split(req.Ref, ":")
	if len(ref) != 2 || ref[1] == "" { //not found if there isn't a ref defined, prevents server crash on sending empty request.
		ctx.Error("Page Not Found", fasthttp.StatusNotFound)
		ctx.Logger().Printf("Error: No reference genotype selected  \n")
		return
	}
	vnt := make(map[string][]string, len(req.Variant))
	vntOrder := make(map[int][]string, len(req.Variant))
	for i := range req.Variant {
		vt := strings.Split(req.Variant[i], ":")
		if len(vt) != 2 || vt[1] == "" { //ignore empty variant
			continue
		}
		if _, ok := vnt[vt[0]]; !ok {
			vnt[vt[0]] = []string{vt[1]}
		} else {
			vnt[vt[0]] = append(vnt[vt[0]], vt[1])
		}
		vntOrder[i] = []string{vt[0], vt[1]}
	}

	// populate experiment if possible due to auth state
	var expSet DataFiles
	if experiments[ref[0]].Location != "" {
		expSet = experiments[ref[0]]
	} else if authState := ctx.UserValue("auth"); authState != nil && privateExp[(authState).(string)][ref[0]].Location != "" {
		expSet = privateExp[(authState).(string)][ref[0]]
	} else {
		ctx.Error("Experiment Not Available", fasthttp.StatusUnauthorized)
		ctx.Logger().Printf("Cancel request for %s - %dns - Invalid credentials or non-extant dataset", ctx.PostArgs(), time.Now().Sub(start).Nanoseconds())
		return
	}

	r, err := ReadFile(expSet.Location, expSet.Gzip)
	if err != nil {
		ctx.Error("Problem reading reference genotype's file: %s \n", fasthttp.StatusInternalServerError)
		ctx.Logger().Printf("Error: Problem reading reference genotype's file: %s", err)
		return
	}

	var b bytes.Buffer
	writer, err := gff.NewWriter(&b)

	if err != nil {
		ctx.Error("Problem opening gff writer: %s \n", fasthttp.StatusInternalServerError)
		ctx.Logger().Printf("Error: Problem opening gff writer: %s", err)
		return
	}

	ctg := make(map[string]int)
	for i := range r.Header.Contigs {
		ctgLen, _ := strconv.Atoi(r.Header.Contigs[i].Optional["length"])
		ctg[r.Header.Contigs[i].Id] = ctgLen
	}

	sameCtr := make(map[string]int, len(vnt[ref[0]])+1)
	diffCtr := make(map[string]int, len(vnt[ref[0]])+1)
	totalCtr := make(map[string]int, len(vnt[ref[0]])+3)
	totalCtr[ref[1]] = 0
	totalCtr["undefined"] = 0
	totalCtr["value"] = 0
	sameCtr["value"] = 0
	diffCtr["value"] = 0

	for i := range vnt[ref[0]] {
		gt := vnt[ref[0]][i]
		sameCtr[gt] = 0
		diffCtr[gt] = 0
		totalCtr[gt] = 0
	}

	var feat *vcf.Feature
	var readErr error
	var contig string
	var stepSize int
	if source == "" || binSize == 0 {
		SetDefaults() // if somehow the source hasn't been set yet
	}
	if req.Bin > 0 {
		stepSize = req.Bin
	} else {
		stepSize = binSize
	}

	stepCt := 0
	stepVal := 0

	for readErr == nil {
		feat, readErr = r.Read()
		if feat != nil {
			gt, _ := feat.SingleGenotype(ref[1], r.Header.Genotypes)
			rt, _ := feat.MultipleGenotypes(vnt[ref[0]], r.Header.Genotypes)
			// reset contig based features, assuming that file is sorted by contig and ascending position
			// when contig changes or you step outside of current bin
			if feat.Pos > uint64(stepCt*stepVal) || contig != feat.Chrom {
				if stepCt > 0 {
					end := uint64(stepCt) * uint64(stepVal)
					if ctg[contig] > 0 && end > uint64(ctg[contig]) {
						end = uint64(ctg[contig])
					}

					printGffLine(writer, contig, stepCt, stepVal, end, sameCtr, diffCtr, totalCtr)

					//Reset counters
					for val := range totalCtr {
						totalCtr[val] = 0
					}
					for val := range sameCtr {
						sameCtr[val] = 0
					}
					for val := range diffCtr {
						diffCtr[val] = 0
					}
					stepCt = (int(feat.Pos) / stepSize) + 1
				}

				if contig != feat.Chrom {
					contig = feat.Chrom
					if ctg[contig] > 0 {
						stepVal = int(float64(ctg[contig]) / math.Ceil(float64(ctg[contig])/float64(stepSize)))
					} else {
						stepVal = stepSize
					}
					stepCt = 1
				}
			}
			gFields := gt.Fields["GT"]
			if gFields != "./." && gFields != ".|." {
				totalCtr["value"]++
				totalCtr[ref[1]]++

				for i := range rt {
					rFields := rt[i].Fields["GT"]
					id := rt[i].Id
					if rFields == "./." || rFields == ".|." {
						totalCtr["undefined"]++
					} else if gFields == rFields {
						sameCtr[id]++
						sameCtr["value"]++
						totalCtr[id]++
					} else {
						diffCtr[id]++
						diffCtr["value"]++
						totalCtr[id]++
					}
				}
			}
		}
	}

	end := uint64(stepCt) * uint64(stepVal)

	if ctg[contig] > 0 && end > uint64(ctg[contig]) {
		end = uint64(ctg[contig])
	}

	printGffLine(writer, contig, stepCt, stepVal, end, sameCtr, diffCtr, totalCtr)

	//send build gff
	ctx.SetContentType("text/plain; charset=utf8")
	fmt.Fprintf(ctx, "%s", b.String())
	//Log completed request
	ctx.Logger().Printf("Return request for %s - %dns", ctx.PostArgs(), time.Now().Sub(start).Nanoseconds())
}

// Auth stuff

func BasicAuth(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		// Get the Basic Authentication credentials
		user, ok := basicAuth(ctx)
		if ok {
			ctx.SetUserValue("auth", user)
		}
		h(ctx)
		return
	})
}

func basicAuth(ctx *fasthttp.RequestCtx) (username string, ok bool) {
	// check for auth header
	auth := ctx.Request.Header.Peek("Authorization")
	if auth == nil {
		return
	}
	// check that auth is basic auth
	sauth := string(auth)
	prefix := "Basic "
	if !strings.HasPrefix(sauth, prefix) {
		return
	}
	// decode authstring
	dec, err := base64.StdEncoding.DecodeString(sauth[len(prefix):])
	if err != nil {
		return
	}
	// find where username:password splits
	sdec := string(dec)
	s := strings.IndexByte(sdec, ':')
	if s < 0 {
		return
	}

	// set user and password
	user := sdec[:s]
	pw := sdec[s+1:]

	// check if user : password is correct
	var C map[string]interface{}
	_ = viper.Unmarshal(&C)
	authUsers := viper.Sub("users")
	pss := authUsers.GetString(user)
	status := pss != "" && pss == pw

	return user, status
}