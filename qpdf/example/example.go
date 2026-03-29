package main

import (
	"github.com/google/uuid"

	// "github.com/hsuanshao/golibs/bucket"
	// bkte "github.com/hsuanshao/golibs/bucket/entity"
	"github.com/hsuanshao/golibs/ctx"
	"github.com/hsuanshao/golibs/qpdf"
)

func main() {
	ctx := ctx.Background()
	pdflib := qpdf.New()
	pdflib.Initial(ctx, qpdf.Landscape, qpdf.A4)

	// use our cloud bucket service
	//bktSrv := bucket.New(nil, "ap-south-1", "dev.pqscale.fileuploads")

	// headerTable := qpdf.HeaderTable{
	// 	HasLogo: true,
	// 	Texts:   []string{"PQScale Usage Report demo", "test1", "test2"},
	// }

	// pdflib.SetHeader(ctx, &headerTable)

	absTab := qpdf.AbstractTable{
		Rows: []*qpdf.AbstractRow{
			{
				Title:   "Aggregation Signature usage report",
				Content: "2023/01 Massa Aggregation Signature Usage Report",
			},
			{
				Title:   "Generate Date",
				Content: "2023/02/01",
			},
		},
	}
	pdflib.AppendAbstractTable(ctx, &absTab)

	dataTable := qpdf.TableForm{
		Rows: []*qpdf.TableRow{
			{
				IsHeader: true,
				Cells: []*qpdf.TableCell{
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "Request Date",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "Request Time",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "Batch Size",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "Token Name",
					},
					{
						Width:   2,
						Align:   qpdf.Center,
						Content: "Instance Code",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "CPU",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "Memory Usage",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "FPGAs",
					},
					{
						Width:   1,
						Align:   qpdf.Center,
						Content: "result",
					},
				},
			},
			{
				Cells: []*qpdf.TableCell{
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "2023/02/1",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "7:00:00",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "1722",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "FPGA4x",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "EC2_F1_T",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "16",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "384",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "2*F1",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "success",
					},
				},
			},
			{
				Cells: []*qpdf.TableCell{
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "2023/02/1",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "7:04:30",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "2722",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "FPGA4x",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "EC2_F1_T",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "16",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "599",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "2*F1",
					},
					{
						Width:   1,
						Align:   qpdf.Left,
						Content: "success",
					},
				},
			},
		},
	}

	pdflib.AppendTable(ctx, &dataTable)

	tmpfilename := uuid.New().String() + "_testing.pdf"

	// NOTE example, upload pdf file to cloud bucket
	// bytes, _ := pdflib.GetPDFbyte(ctx)
	// bktWriter := bktSrv.Writer(ctx)
	// objURL, err := bktWriter.Upload(ctx, bkte.PDF, tmpfilename, bytes)
	// if err != nil {
	// 	ctx.WithFields(logrus.Fields{"err": err, "path": tmpfilename}).Error("upload file to s3 failed")
	// 	return
	// }

	// bktReader := bktSrv.Reader(ctx)
	// duration := 10 * time.Minute
	// presignedURL, err := bktReader.GetPresignedURL(ctx, &duration, objURL)
	// if err != nil {
	// 	ctx.WithFields(logrus.Fields{"err": err, "duration": duration, "objURL": objURL}).Error("get presigned url fail")
	// 	return
	// }
	// ctx.WithField("presignedURL", presignedURL).Info("check presigned url")

	pdflib.Save(ctx, tmpfilename)
}
