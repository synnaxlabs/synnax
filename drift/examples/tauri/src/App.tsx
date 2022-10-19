import reactLogo from './assets/react.svg';
import { useSelector, useDispatch } from 'react-redux';
import './App.css';
import store, { incremented } from './store';
import { createWindow } from '@synnaxlabs/drift';
import { appWindow } from '@tauri-apps/api/window';

function App() {
  const { value: count } = useSelector((state: any) => state.counter);
  const dispatch = useDispatch();
  console.log(appWindow.label);
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
            dispatch(
              createWindow({
                key: `count-${count}`,
                url: 'http://localhost:5173',
              })
            );
          }}
        >
          count is {count}
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
