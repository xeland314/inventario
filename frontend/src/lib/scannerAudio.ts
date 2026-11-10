/**
 * Efectos de sonido POS para escáner Symbol USB mediante Web Audio API.
 * Emula el clásico pitido limpio de caja registradora / terminal óptica (1950 Hz)
 * y un acorde ascendente de doble tono cuando hay coincidencia de inventario.
 */

export class ScannerSoundEffects {
  private ctx: AudioContext | null = null;
  public enabled: boolean;

  constructor() {
    this.enabled = typeof window !== 'undefined' ? localStorage.getItem('scanner_sound') !== 'false' : true;
  }

  private init(): void {
    if (typeof window === 'undefined') return;
    if (!this.ctx) {
      const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
      if (AudioCtx) {
        this.ctx = new AudioCtx();
      }
    }
    if (this.ctx && this.ctx.state === 'suspended') {
      void this.ctx.resume();
    }
  }

  public toggle(): boolean {
    this.enabled = !this.enabled;
    if (typeof window !== 'undefined') {
      localStorage.setItem('scanner_sound', this.enabled ? 'true' : 'false');
    }
    if (this.enabled) {
      this.playSuccess();
    }
    return this.enabled;
  }

  /**
   * Pitido clásico y nítido de escáner Symbol (1950 Hz, 80ms)
   */
  public playSuccess(): void {
    if (!this.enabled) return;
    this.init();
    if (!this.ctx) return;

    try {
      const now = this.ctx.currentTime;
      const osc = this.ctx.createOscillator();
      const gain = this.ctx.createGain();

      osc.type = 'sine';
      osc.frequency.setValueAtTime(1950, now);

      gain.gain.setValueAtTime(0.28, now);
      gain.gain.exponentialRampToValueAtTime(0.001, now + 0.08);

      osc.connect(gain);
      gain.connect(this.ctx.destination);

      osc.start(now);
      osc.stop(now + 0.08);
    } catch (e) {
      console.warn('Audio no disponible:', e);
    }
  }

  /**
   * Acorde ascendente dos tonos (1800Hz -> 2400Hz, 110ms) cuando se encuentra un producto en inventario
   */
  public playMatch(): void {
    if (!this.enabled) return;
    this.init();
    if (!this.ctx) return;

    try {
      const now = this.ctx.currentTime;
      const osc = this.ctx.createOscillator();
      const gain = this.ctx.createGain();

      osc.type = 'sine';
      osc.frequency.setValueAtTime(1800, now);
      osc.frequency.setValueAtTime(2400, now + 0.04);

      gain.gain.setValueAtTime(0.32, now);
      gain.gain.exponentialRampToValueAtTime(0.001, now + 0.11);

      osc.connect(gain);
      gain.connect(this.ctx.destination);

      osc.start(now);
      osc.stop(now + 0.11);
    } catch (e) {
      console.warn('Audio no disponible:', e);
    }
  }
}

export const scannerSounds = new ScannerSoundEffects();
