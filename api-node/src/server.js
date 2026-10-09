import { createApp } from './app.js';
import { config } from './config/env.js';

createApp().listen(config.port, () => {
  console.log(`Matrix statistics service listening on port ${config.port}`);
});
