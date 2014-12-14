var React = require('react/addons')
var Reflux = require('reflux')
var cx = React.addons.classSet

var actions = require('../actions')
var Scroller = require('./scroller')
var Chat = require('./chat')
var NotifyToggle = require('./notifytoggle')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat').store, 'chat'),
  ],

  getInitialState: function() {
    return {formFocus: false}
  },

  send: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.sendMessage(input.value)
    input.value = ''
    ev.preventDefault()
  },

  setNick: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.setNick(input.value)
    ev.preventDefault()
  },

  focusInput: function() {
    this.refs.input.getDOMNode().focus()
  },

  onFormFocus: function() {
    this.setState({formFocus: true})
  },

  onFormBlur: function() {
    this.setState({formFocus: false})
  },

  render: function() {
    var sendForm
    if (this.state.chat.nick) {
      sendForm = (
        <form onSubmit={this.send} className={cx({'focus': this.state.formFocus})}>
          <div className="nick-box">
            <span className="nick">{this.state.chat.nick}</span>
          </div>
          <input key="msg" ref="input" type="text" autoFocus onFocus={this.onFormFocus} onBlur={this.onFormBlur} />
        </form>
      )
    } else {
      sendForm = (
        <form onSubmit={this.setNick} className={cx({'focus': this.state.formFocus})}>
          <label>choose a nickname to start chatting:</label>
          <input key="nick" ref="input" type="text" onFocus={this.onFormFocus} onBlur={this.onFormBlur} />
        </form>
      )
    }

    return (
      <div className="chat">
        <Scroller className="messages-container" onClick={this.focusInput}>
          <div className="messages-content">
            {sendForm}
            <Chat messages={this.state.chat.messages} />
            <div className="overlay">
              <div className="options">
                <NotifyToggle />
              </div>
              <div className={cx({'status': true, 'disconnected': this.state.chat.connected == false})}>disconnected!</div>
            </div>
          </div>
        </Scroller>
      </div>
    )
  },
})
