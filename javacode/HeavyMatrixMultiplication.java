package javacode;
 import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

public class HeavyMatrixMultiplication {

    public static void main(String[] args) throws InterruptedException {
        int size = 2000; // size of matrix: size x size
        double[][] A = new double[size][size];
        double[][] B = new double[size][size];
        double[][] C = new double[size][size];

        // Initialize matrices with random values
        for (int i = 0; i < size; i++) {
            for (int j = 0; j < size; j++) {
                A[i][j] = Math.random();
                B[i][j] = Math.random();
            }
        }

        long start = System.currentTimeMillis();

        // Use 2 threads for parallel computation
        int cores = 2;
        ExecutorService executor = Executors.newFixedThreadPool(cores);

        for (int i = 0; i < size; i++) {
            final int row = i;
            executor.submit(() -> {
                for (int j = 0; j < size; j++) {
                    double sum = 0;
                    for (int k = 0; k < size; k++) {
                        sum += A[row][k] * B[k][j];
                    }
                    C[row][j] = sum;
                }
            });
        }

        executor.shutdown();
        executor.awaitTermination(1, TimeUnit.HOURS);

        long end = System.currentTimeMillis();
        System.out.println("Matrix multiplication done in: " + (end - start) + " ms");
        System.out.println("C[0][0] = " + C[0][0]); // just to prevent optimization
    }
}
