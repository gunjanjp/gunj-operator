/**
 * BackpressureManager.ts
 * Manages backpressure and rate limiting for real-time data streams
 */

export interface Message {
  id: string;
  type: string;
  data: any;
  priority: number;
  timestamp: Date;
}

export interface PerformanceMetrics {
  cpuUsage: number;
  memoryUsage: number;
  frameRate: number;
  eventLoopDelay: number;
}

/**
 * Priority queue implementation for message ordering
 */
class PriorityQueue<T> {
  private items: Array<{ element: T; priority: number }> = [];

  enqueue(element: T, priority: number): void {
    const queueElement = { element, priority };
    let added = false;

    for (let i = 0; i < this.items.length; i++) {
      if (queueElement.priority > this.items[i].priority) {
        this.items.splice(i, 0, queueElement);
        added = true;
        break;
      }
    }

    if (!added) {
      this.items.push(queueElement);
    }
  }

  dequeue(): T | undefined {
    return this.items.shift()?.element;
  }

  isEmpty(): boolean {
    return this.items.length === 0;
  }

  size(): number {
    return this.items.length;
  }

  clear(): void {
    this.items = [];
  }

  peek(): T | undefined {
    return this.items[0]?.element;
  }
}

export class BackpressureManager {
  private messageQueue: PriorityQueue<Message> = new PriorityQueue();
  private processingRate: number = 100; // messages per second
  private bufferSize: number = 10000;
  private dropPolicy: 'oldest' | 'lowest-priority' = 'oldest';
  private isProcessing: boolean = false;
  private droppedMessages: number = 0;
  private processedMessages: number = 0;
  private lastPerformanceCheck: Date = new Date();
  private performanceCheckInterval: number = 5000; // 5 seconds
  private messageHandlers: Map<string, (message: Message) => Promise<void>> = new Map();

  constructor(config?: {
    processingRate?: number;
    bufferSize?: number;
    dropPolicy?: 'oldest' | 'lowest-priority';
  }) {
    if (config?.processingRate) this.processingRate = config.processingRate;
    if (config?.bufferSize) this.bufferSize = config.bufferSize;
    if (config?.dropPolicy) this.dropPolicy = config.dropPolicy;

    // Start performance monitoring
    this.startPerformanceMonitoring();
  }

  /**
   * Register a message handler for a specific message type
   */
  registerHandler(messageType: string, handler: (message: Message) => Promise<void>): void {
    this.messageHandlers.set(messageType, handler);
  }

  /**
   * Process a message with backpressure handling
   */
  async processMessage(message: Message): Promise<void> {
    // Check buffer capacity
    if (this.messageQueue.size() >= this.bufferSize) {
      this.handleBufferOverflow();
    }

    // Add to queue with priority
    this.messageQueue.enqueue(message, message.priority);

    // Start processing if not already running
    if (!this.isProcessing) {
      this.startProcessing();
    }
  }

  /**
   * Handle buffer overflow based on drop policy
   */
  private handleBufferOverflow(): void {
    if (this.dropPolicy === 'oldest') {
      // Drop oldest messages from the front
      const dropCount = Math.floor(this.bufferSize * 0.1); // Drop 10%
      for (let i = 0; i < dropCount; i++) {
        this.messageQueue.dequeue();
        this.droppedMessages++;
      }
    } else {
      // Drop lowest priority messages
      // This would require a more complex implementation
      // For now, just drop from the end
      const currentSize = this.messageQueue.size();
      const targetSize = Math.floor(this.bufferSize * 0.9);
      const dropCount = currentSize - targetSize;
      
      // Clear and rebuild queue without lowest priority items
      const tempItems: Array<{ message: Message; priority: number }> = [];
      while (!this.messageQueue.isEmpty()) {
        const message = this.messageQueue.dequeue();
        if (message) {
          tempItems.push({ message, priority: message.priority });
        }
      }
      
      // Sort by priority and keep only highest priority items
      tempItems.sort((a, b) => b.priority - a.priority);
      tempItems.slice(0, targetSize).forEach(item => {
        this.messageQueue.enqueue(item.message, item.priority);
      });
      
      this.droppedMessages += dropCount;
    }

    console.warn(`Buffer overflow: dropped ${this.droppedMessages} messages`);
  }

  /**
   * Start processing messages with rate limiting
   */
  private async startProcessing(): Promise<void> {
    if (this.isProcessing) return;
    
    this.isProcessing = true;
    const delay = 1000 / this.processingRate;

    while (!this.messageQueue.isEmpty()) {
      const startTime = Date.now();
      const message = this.messageQueue.dequeue();

      if (message) {
        try {
          await this.handleMessage(message);
          this.processedMessages++;
        } catch (error) {
          this.handleProcessingError(error, message);
        }
      }

      // Enforce rate limit
      const processingTime = Date.now() - startTime;
      const sleepTime = Math.max(0, delay - processingTime);
      
      if (sleepTime > 0) {
        await this.sleep(sleepTime);
      }

      // Check if we should adjust processing rate
      if (Date.now() - this.lastPerformanceCheck.getTime() > this.performanceCheckInterval) {
        this.checkPerformance();
      }
    }

    this.isProcessing = false;
  }

  /**
   * Handle individual message
   */
  private async handleMessage(message: Message): Promise<void> {
    const handler = this.messageHandlers.get(message.type);
    
    if (handler) {
      await handler(message);
    } else {
      console.warn(`No handler registered for message type: ${message.type}`);
    }
  }

  /**
   * Handle processing errors
   */
  private handleProcessingError(error: any, message: Message): void {
    console.error(`Error processing message ${message.id}:`, error);
    
    // Could implement retry logic here
    // For now, just log and continue
  }

  /**
   * Sleep for specified milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Start performance monitoring
   */
  private startPerformanceMonitoring(): void {
    // Monitor frame rate
    let lastTime = performance.now();
    let frames = 0;
    
    const measureFPS = () => {
      frames++;
      const currentTime = performance.now();
      
      if (currentTime >= lastTime + 1000) {
        const fps = Math.round((frames * 1000) / (currentTime - lastTime));
        frames = 0;
        lastTime = currentTime;
        
        // Store FPS for performance checks
        (window as any).__backpressureFPS = fps;
      }
      
      requestAnimationFrame(measureFPS);
    };
    
    requestAnimationFrame(measureFPS);
  }

  /**
   * Check performance and adjust processing rate
   */
  private checkPerformance(): void {
    const metrics = this.getPerformanceMetrics();
    this.adjustProcessingRate(metrics);
    this.lastPerformanceCheck = new Date();
  }

  /**
   * Get current performance metrics
   */
  private getPerformanceMetrics(): PerformanceMetrics {
    // Get memory usage
    const memoryUsage = (performance as any).memory
      ? ((performance as any).memory.usedJSHeapSize / (performance as any).memory.jsHeapSizeLimit) * 100
      : 50; // Default to 50% if not available

    // Get frame rate
    const frameRate = (window as any).__backpressureFPS || 60;

    // Estimate CPU usage based on processing speed
    const expectedProcessingTime = 1000 / this.processingRate;
    const actualProcessingTime = this.getAverageProcessingTime();
    const cpuUsage = Math.min(100, (actualProcessingTime / expectedProcessingTime) * 100);

    // Event loop delay (simplified)
    const eventLoopDelay = Math.max(0, this.messageQueue.size() / this.processingRate * 1000);

    return {
      cpuUsage,
      memoryUsage,
      frameRate,
      eventLoopDelay,
    };
  }

  /**
   * Get average processing time per message
   */
  private getAverageProcessingTime(): number {
    // This is a simplified implementation
    // In reality, you'd track actual processing times
    return 1000 / this.processingRate * 0.8; // Assume 80% efficiency
  }

  /**
   * Adjust processing rate based on performance metrics
   */
  adjustProcessingRate(metrics: PerformanceMetrics): void {
    const oldRate = this.processingRate;

    // Reduce rate if performance is degraded
    if (metrics.cpuUsage > 80 || metrics.memoryUsage > 85 || metrics.frameRate < 30) {
      this.processingRate = Math.max(50, Math.floor(this.processingRate * 0.8));
    }
    // Increase rate if performance is good
    else if (metrics.cpuUsage < 50 && metrics.memoryUsage < 60 && metrics.frameRate > 50) {
      this.processingRate = Math.min(200, Math.floor(this.processingRate * 1.2));
    }

    if (oldRate !== this.processingRate) {
      console.log(`Adjusted processing rate from ${oldRate} to ${this.processingRate} msgs/sec`);
    }
  }

  /**
   * Get current statistics
   */
  getStats(): {
    queueSize: number;
    processingRate: number;
    processedMessages: number;
    droppedMessages: number;
    bufferUtilization: number;
  } {
    return {
      queueSize: this.messageQueue.size(),
      processingRate: this.processingRate,
      processedMessages: this.processedMessages,
      droppedMessages: this.droppedMessages,
      bufferUtilization: (this.messageQueue.size() / this.bufferSize) * 100,
    };
  }

  /**
   * Clear the message queue
   */
  clear(): void {
    this.messageQueue.clear();
    this.droppedMessages = 0;
    this.processedMessages = 0;
  }

  /**
   * Stop processing
   */
  stop(): void {
    this.isProcessing = false;
    this.clear();
  }
}
