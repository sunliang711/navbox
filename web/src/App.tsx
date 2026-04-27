import { Boxes } from 'lucide-react';
import './styles.css';

export function App() {
  return (
    <main className="app-shell">
      <section className="hero">
        <div className="brand-mark">
          <Boxes size={32} aria-hidden="true" />
        </div>
        <div>
          <h1>Navbox</h1>
          <p>Tag based navigation is ready for the next implementation task.</p>
        </div>
      </section>
    </main>
  );
}
