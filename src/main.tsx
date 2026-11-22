import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import { ErrorBoundary } from './components/ErrorBoundary';
import './styles.css';

// Catch uncaught runtime errors outside React tree so the window never silently goes blank
window.addEventListener('error', (ev) => {
  const root = document.getElementById('root');
  if (root && root.innerHTML.trim() === '') {
    root.innerHTML = `<div style="padding:32px;font-family:-apple-system,sans-serif;color:#b91c1c"><h3>Error</h3><pre>${String(ev.message || ev.error)}</pre></div>`;
  }
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
);
