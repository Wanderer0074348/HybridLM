import json
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.gridspec as gridspec

# ── Load stats ──────────────────────────────────────────────────────────────
def load(path):
    with open(path) as f:
        d = json.load(f)
    return d['true_positives'], d['true_negatives'], d['false_positives'], d['false_negatives'], \
           d['routing_accuracy_pct'], d['f1_score']

rb = load('results_custom_rulebased.json')
ml = load('results_custom_ML_only.json')
rl = load('results_custom_MLRL.json')

configs = [
    (rb, 'Rule-Based (Fallback)'),
    (ml, 'ML-Only (Logistic Reg)'),
    (rl, 'ML + RL (Hybrid)'),
]

TOTAL = 100

# ── Figure ───────────────────────────────────────────────────────────────────
fig, axes = plt.subplots(1, 3, figsize=(14, 4.5))
fig.suptitle(
    'Confusion Matrix Comparison – Routing Strategies (100 Test Queries)',
    fontsize=13, fontweight='bold', y=1.02
)

CMAP = 'Blues'

for ax, ((tp, tn, fp, fn, acc, f1), title) in zip(axes, configs):
    # matrix rows = Actual (LLM, SLM), cols = Predicted (LLM, SLM)
    mat = np.array([[tp, fn],
                    [fp, tn]], dtype=float)

    im = ax.imshow(mat, cmap=CMAP, vmin=0, vmax=TOTAL * 0.6, aspect='auto')
    plt.colorbar(im, ax=ax, fraction=0.046, pad=0.04)

    # Tick labels
    ax.set_xticks([0, 1])
    ax.set_xticklabels(['Predicted\nLLM', 'Predicted\nSLM'], fontsize=10)
    ax.set_yticks([0, 1])
    ax.set_yticklabels(['Actual\nLLM', 'Actual\nSLM'], fontsize=10, rotation=90, va='center')
    ax.set_xlabel('Predicted Label', fontsize=10)
    ax.set_ylabel('Actual Label', fontsize=10)

    # Cell annotations
    cell_labels = [
        [f'{tp}\n({tp/TOTAL*100:.1f}%)', f'{fn}\n({fn/TOTAL*100:.1f}%)'],
        [f'{fp}\n({fp/TOTAL*100:.1f}%)', f'{tn}\n({tn/TOTAL*100:.1f}%)'],
    ]
    threshold = mat.max() * 0.55
    for i in range(2):
        for j in range(2):
            color = 'white' if mat[i, j] > threshold else 'black'
            ax.text(j, i, cell_labels[i][j], ha='center', va='center',
                    fontsize=12, fontweight='bold', color=color)

    ax.set_title(f'{title}\nAccuracy: {acc:.1f}% | F1: {f1:.2f}',
                 fontsize=11, pad=8)

plt.tight_layout()
plt.savefig('confusion_matrices.png', dpi=180, bbox_inches='tight')
print("Saved confusion_matrices.png")
plt.show()
