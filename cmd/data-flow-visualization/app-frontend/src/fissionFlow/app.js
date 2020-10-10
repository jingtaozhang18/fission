import FlowPlot from './flowGraph'
import React, {Component} from "react";
import getFlowData from './dataApi'
import Style from './app.css'
import Form from "./Form";

class app extends Component {
    state = {flowData: {}}

    updateFlow = (time, step) => {
        getFlowData(time, step).then(data => {
            this.setState({
                flowData: data
            })
        })
    }

    constructor(props) {
        super(props);
        this.updateFlow("", "")
        // this.timer = setInterval(() => this.updateFlow(), 2000)
    }


    render() {

        return (
            <div className={Style.flexMainBody}>
                <Form handleSubmit={this.updateFlow}/>
                <FlowPlot data={this.state.flowData}/>
            </div>
        )
    }
}

export default app