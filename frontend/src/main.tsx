import { hydrate } from 'preact-iso'
import { App } from './App'
import './styles.css'
import './app.css'

hydrate(<App />, document.getElementById('app')!)
