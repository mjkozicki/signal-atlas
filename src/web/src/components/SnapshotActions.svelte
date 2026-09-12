<script lang="ts">
  import { Eye, EyeOff, Trash2 } from '@lucide/svelte';
  let { id, label, hidden = false, disabled = false, onaction }: {
    id: string; label: string; hidden?: boolean; disabled?: boolean;
    onaction: (action: 'hide' | 'restore' | 'delete') => void;
  } = $props();
  function remove() {
    if (window.confirm(`Permanently delete this snapshot?\n\n${label}\n${id}\n\nThis removes its saved observations and findings from this service. It cannot be undone. Exported files and original captures are not deleted.`)) onaction('delete');
  }
</script>
<div class="snapshot-actions" aria-label={`Actions for snapshot ${id}`}>
  {#if hidden}<span class="badge neutral">Hidden</span>{/if}
  <button class="secondary" {disabled} aria-label={hidden ? 'Restore snapshot' : 'Hide snapshot'} title={hidden ? 'Return to default history' : 'Hide from default history; restore using Show hidden snapshots'} onclick={() => onaction(hidden ? 'restore' : 'hide')}>{#if hidden}<Eye size={14}/>{:else}<EyeOff size={14}/>{/if}{hidden ? 'Restore' : 'Hide'}</button>
  <button class="secondary delete-snapshot" {disabled} aria-label="Delete snapshot" title="Permanently delete this saved snapshot" onclick={remove}><Trash2 size={14}/> Delete</button>
</div>
<style>
.snapshot-actions{display:flex;align-items:center;gap:8px;flex-wrap:wrap}.snapshot-actions button{font-size:12px;padding:8px 11px}.snapshot-actions .delete-snapshot{color:#a54146;border-color:#e8cdd0}.snapshot-actions button:disabled{cursor:not-allowed}.snapshot-actions .badge{font-size:11px}
</style>
