import React, {Component} from 'react'
import moment from 'moment';

class Form extends Component {
    initialState = {
        time: '2018-06-12T19:30',
        step: '',
    }

    constructor(props) {
        super(props);
        console.log(moment().format())
        this.initialState.time = moment().format("YYYY-MM-DDTHH:mm")
        this.state = this.initialState
    }


    handleChange = (event) => {
        const {name, value} = event.target
        this.setState({
                [name]: value,
            },
        )
    }

    submitForm = () => {
        console.log(moment(this.state.time, "YYYY-MM-DDTHH:mm").unix(), this.state.step)
        this.props.handleSubmit(moment(this.state.time, "YYYY-MM-DDTHH:mm").unix(), this.state.step)
    }

    render() {
        const {time, step} = this.state;

        return (
            <form>
                <label htmlFor="name">时间</label>
                <input
                    type="datetime-local"
                    name="time"
                    id="time"
                    value={time}
                    onChange={this.handleChange}/>
                <label htmlFor="job">跨度</label>
                <input
                    type="text"
                    name="step"
                    id="step"
                    value={step}
                    onChange={this.handleChange}/>
                <input type="button" value="Submit" onClick={this.submitForm}/>
            </form>
        );
    }
}

export default Form;