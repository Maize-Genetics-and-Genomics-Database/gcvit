import React from 'react';

export default class DataModal extends React.Component {
    state = {
        name: 'cvit',
        format: 'svg',
        quality: .95,
        gffOptions: {
            'chromosome':true,
            'diff':true,
            'same':true,
            'total':true
        }
    };

    exportImage = (blob) => {
        let url = URL.createObjectURL(blob);
        this.saveImage(url);
    };

    saveImage = (url) => {
        let name = this.state.name !== '' ? this.state.name : 'cvit';
        name += `.${this.state.format}`;
        let link = document.createElement('a');
        link.download = name;
        link.href = url;
        document.body.appendChild(link);
        link.click();
    };

    onClickImage = () => {
        let paper = window.cvit.model.paper;
        console.log(paper);
        if(this.state.format === 'svg'){
            let url = 'data:image/svg+xml;utf8,' +
                encodeURIComponent(paper.project.exportSVG({asString:true}));
            this.saveImage(url);
        } else {
            paper.project.view.element.toBlob((blob) => this.exportImage(blob));
        }
    };

    onClickData = () => {
        let gff = '##gff-version 3.2.1';
        let data = window.cvit.model.data;
        let gffOpts = this.state.gffOptions;
        for (let group in data) {
            if(data.hasOwnProperty(group) && gffOpts[group]) {
                let dataGroup = data[group];
                if( dataGroup.hasOwnProperty('features')) {
                    dataGroup.features.forEach(feature => {
                        let line = `${feature.seqName}\t${feature.source}\t${feature.feature}\t${feature.start}\t${feature.end}\t${feature.score}\t${feature.strand}\t${feature.frame}`;
                        let attributes = '';
                        for (let key in feature.attribute) {
                            if (feature.attribute.hasOwnProperty(key)) {
                                attributes += `${key}=${feature.attribute[key]};`
                            }
                        }
                        gff +=`\n${line}\t${attributes}`;
                    });
                }
            }
        }
        let url = 'data:text/plain;utf8,' +
            encodeURIComponent(gff);
        let win = window.open();
        // noinspection HtmlDeprecatedAttribute
        win.document.write('<iframe src="' + url  + '" frameborder="0" style="border:0; top:0; left:0; bottom:0; right:0; width:100%; height:100%;" allowfullscreen></iframe>');
        win.download('gcvit.gff');
        // link.href = url;
        // document.body.appendChild(link);
        // link.click();
    };

    onInput = (evt) =>{
        this.setState({name:evt.target.value});
    };

    onSelect = (evt) => {
        this.setState({format:evt.target.value});
    };

    onChecked = (evt) => {
        let gffOptions = this.state.gffOptions;
        gffOptions[evt.target.value] = !gffOptions[evt.target.value];
        this.setState( {gffOptions});
    };

    render(props,state){
        let {name,format,gffOptions} = this.state;
        return(
            <div className={"modal-area selector-container"}>
                <div className={"modal-content"} >
                    <h5> Downloads </h5>
                    <hr />
                    <div className={'modal-contents'}>
                        <div className={'pure-g'}>
                            <div className={'pure-u-1-1 l-box cvit cvit-modal'} id={'export-modal'} >
                                <h5> Download Image </h5>
                                <p> Download the current view as an image.</p>

                                <form style={{width:'100%'}}>
                                    <h6> Image Settings: </h6>
                                    <tbody>
                                    <tr>
                                        <td><span>File Name: </span></td>
                                        <td>
                                            <input
                                                type={'text'}
                                                value={name}
                                                onInput={(evt)=>this.onInput(evt)}
                                                placeholder={'cvit'}
                                            />
                                        </td>
                                    </tr>
                                    <tr>
                                        <td> <span> File Type: </span> </td>
                                        <td>
                                            <label>
                                                <input
                                                    id={'opt-svg'}
                                                    type={'radio'}
                                                    value={'svg'}
                                                    onChange={(evt)=>this.onSelect(evt)}
                                                    checked={format === 'svg'} />
                                                <span> svg </span>
                                            </label>
                                        </td>
                                        <td>
                                            <label>
                                                <input
                                                    id={'opt-png'}
                                                    type={'radio'}
                                                    value={'png'}
                                                    onChange={(evt)=>this.onSelect(evt)}
                                                    checked={format === 'png'}
                                                />
                                                <span> png </span>
                                            </label>
                                        </td>
                                    </tr>
                                    </tbody>
                                </form>
                                <button className={'pure-button button-display modal-confirm'}
                                        onClick={()=>this.onClickImage()}
                                > Export Image </button>
                            </div>
                        </div>
                        <div className={'pure-g'}>
                            <hr />
                            <div className={'pure-u-1-1 l-box cvit cvit-modal'} id={'export-modal'} >
                                <h5> Download Data </h5>
                                <p> Download data as a gff </p>

                                <form style={{width:'100%'}}>
                                    <h6> GFF Settings: </h6>
                                    <tbody>
                                    <tr>
                                        <td> <span> Include: </span> </td>
                                        <td>
                                            <label>
                                                <input
                                                    id={'opt-chr'}
                                                    type={'checkbox'}
                                                    value={'chromosome'}
                                                    onChange={(evt)=>this.onChecked(evt)}
                                                    checked={gffOptions.chromosome}
                                                />
                                                <span> chromosome </span>
                                            </label>
                                        </td>
                                        <td>
                                            <label>
                                                <input
                                                    id={'opt-diff'}
                                                    type={'checkbox'}
                                                    value={'diff'}
                                                    onChange={(evt)=>this.onChecked(evt)}
                                                    checked={gffOptions.diff}
                                                />
                                                <span> different </span>
                                            </label>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td>  </td>
                                        <td>
                                            <label>
                                                <input
                                                    id={'opt-same'}
                                                    type={'checkbox'}
                                                    value={'same'}
                                                    onChange={(evt)=>this.onChecked(evt)}
                                                    checked={gffOptions.same}
                                                />
                                                <span> same </span>
                                            </label>
                                        </td>
                                        <td>
                                            <label>
                                                <input
                                                    id={'opt-total'}
                                                    type={'checkbox'}
                                                    value={'total'}
                                                    onChange={(evt)=>this.onChecked(evt)}
                                                    checked={gffOptions.total}
                                                />
                                                <span> total </span>
                                            </label>
                                        </td>
                                    </tr>
                                    </tbody>
                                </form>

                                <button className={'pure-button button-display modal-confirm'}
                                        onClick={()=>this.onClickData()}
                                > Download Data </button>
                            </div>
                        </div>
                    </div>
                </div>
                <div className={'modal-close'}>
                    <button className={'pure-button  button-display modal-confirm'}
                            onClick={()=>this.props.closeAction()}
                    > Close </button>
                </div>
            </div>
        );
    }
}