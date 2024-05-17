#include <stdio.h>
#include <stdlib.h>

typedef struct {
  int data;
  struct Node *next;
} Node;

Node *createNode(int data) {
  Node *newNode = (Node *)malloc(sizeof(int));
  newNode->data = data;
  newNode->next = NULL;
  return newNode;
}

int main(void) {
  createNode(43);
  return 0;
}
