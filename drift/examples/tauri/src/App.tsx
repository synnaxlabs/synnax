import reactLogo from './assets/react.svg';
import { useSelector, useDispatch } from 'react-redux';
import './App.css';
import { incremented } from './store';
import { createWindow } from '@synnaxlabs/drift';

function App() {
  const { value: count } = useSelector((state: any) => state.counter);
  const { numCreated, windows } = useSelector((state: any) => state.drift);
  const dispatch = useDispatch();
  return (
    <div className="App">
      <div>
        <a href="https://vitejs.dev" target="_blank">
          <img src="/vite.svg" className="logo" alt="Vite logo" />
        </a>
        <a href="https://reactjs.org" target="_blank">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1>Vite + React</h1>
      <div className="card">
        <button
          onClick={() => {
            dispatch(incremented());
          }}
        >
          count is {count}
        </button>
        <button
          onClick={() => {
            dispatch(
              createWindow({
                title: `Window ${numCreated}`,
                url: 'http://localhost:5173',
              })
            );
          }}
        >
          {numCreated} windows created, {Object.keys(windows).length} windows
          open
        </button>
        <p>
          Edit <code>src/App.tsx</code> and save to test HMR
        </p>
      </div>
      <p className="read-the-docs">
        Click on the Vite and React logos to learn more
      </p>
    </div>
  );
}

export default App;
